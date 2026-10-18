package services

import (
	"time"

	"xorm.io/xorm"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

// BookService represents shared book service
type BookService struct {
	ServiceUsingDB
	ServiceUsingUuid
}

// Initialize a shared book service singleton instance
var (
	Books = &BookService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
		ServiceUsingUuid: ServiceUsingUuid{
			container: uuid.Container,
		},
	}
)

// GetSharedBookById returns the shared book of the given id
func (s *BookService) GetSharedBookById(c core.Context, bookId int64) (*models.SharedBook, error) {
	if bookId <= 0 {
		return nil, errs.ErrBookIdInvalid
	}

	book := &models.SharedBook{}
	has, err := s.UserDB().NewSession(c).ID(bookId).Where("deleted=?", false).Get(book)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, errs.ErrSharedBookNotFound
	}

	return book, nil
}

// GetAccessibleSharedBooks returns all shared books which the given user owns or is a member of
func (s *BookService) GetAccessibleSharedBooks(c core.Context, uid int64) ([]*models.SharedBook, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var memberships []*models.BookMember
	err := s.UserDB().NewSession(c).Where("member_uid=?", uid).Find(&memberships)

	if err != nil {
		return nil, err
	}

	bookIds := make([]int64, len(memberships))

	for i := 0; i < len(memberships); i++ {
		bookIds[i] = memberships[i].BookUid
	}

	var books []*models.SharedBook
	sess := s.UserDB().NewSession(c).Where("deleted=?", false)

	if len(bookIds) > 0 {
		condition, args := buildOwnerOrBookIdsCondition(uid, bookIds)
		sess = sess.And(condition, args...)
	} else {
		sess = sess.And("owner_uid=?", uid)
	}

	err = sess.OrderBy("created_unix_time asc").Find(&books)

	return books, err
}

func buildOwnerOrBookIdsCondition(uid int64, bookIds []int64) (string, []any) {
	condition := "(owner_uid=? OR book_id IN ("
	args := make([]any, 0, len(bookIds)+1)
	args = append(args, uid)

	for i := 0; i < len(bookIds); i++ {
		if i > 0 {
			condition += ","
		}

		condition += "?"
		args = append(args, bookIds[i])
	}

	condition += "))"
	return condition, args
}

// CreateSharedBook creates a new shared book owned by the given user
func (s *BookService) CreateSharedBook(c core.Context, ownerUid int64, name string) (*models.SharedBook, error) {
	if ownerUid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	now := time.Now().Unix()
	book := &models.SharedBook{
		BookId:          s.GenerateUuid(uuid.UUID_TYPE_USER),
		OwnerUid:        ownerUid,
		Name:            name,
		CreatedUnixTime: now,
		UpdatedUnixTime: now,
	}

	if book.BookId < 1 {
		return nil, errs.ErrSystemIsBusy
	}

	err := s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		_, err := sess.Insert(book)
		return err
	})

	if err != nil {
		return nil, err
	}

	return book, nil
}

// ModifySharedBookName renames a shared book owned by the given user
func (s *BookService) ModifySharedBookName(c core.Context, ownerUid int64, bookId int64, name string) error {
	if ownerUid <= 0 {
		return errs.ErrUserIdInvalid
	}

	book, err := s.GetSharedBookById(c, bookId)

	if err != nil {
		return err
	} else if book.OwnerUid != ownerUid {
		return errs.ErrOnlyBookOwnerCanModifyBook
	}

	book.Name = name
	book.UpdatedUnixTime = time.Now().Unix()

	return s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		_, err := sess.ID(bookId).Cols("name", "updated_unix_time").Update(book)
		return err
	})
}

// DeleteSharedBook deletes an empty shared book owned by the given user, together with its members and shared accounts
func (s *BookService) DeleteSharedBook(c core.Context, ownerUid int64, bookId int64) error {
	if ownerUid <= 0 {
		return errs.ErrUserIdInvalid
	}

	book, err := s.GetSharedBookById(c, bookId)

	if err != nil {
		return err
	} else if book.OwnerUid != ownerUid {
		return errs.ErrOnlyBookOwnerCanModifyBook
	}

	transactionExists, err := s.UserDataDB(bookId).NewSession(c).Where("uid=? AND deleted=?", bookId, false).Exist(&models.Transaction{})

	if err != nil {
		return err
	} else if transactionExists {
		return errs.ErrSharedBookNotEmpty
	}

	now := time.Now().Unix()

	return s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		updateModel := &models.SharedBook{
			Deleted:         true,
			DeletedUnixTime: now,
		}

		_, err := sess.ID(bookId).Cols("deleted", "deleted_unix_time").Update(updateModel)

		if err != nil {
			return err
		}

		_, err = sess.Where("book_uid=?", bookId).Delete(&models.BookMember{})

		if err != nil {
			return err
		}

		_, err = sess.Where("book_id=?", bookId).Delete(&models.SharedBookAccount{})
		return err
	})
}

// GetBookMembers returns all members (excluding the owner) of the given shared book
func (s *BookService) GetBookMembers(c core.Context, bookId int64) ([]*models.BookMember, error) {
	if bookId <= 0 {
		return nil, errs.ErrBookIdInvalid
	}

	var members []*models.BookMember
	err := s.UserDB().NewSession(c).Where("book_uid=?", bookId).OrderBy("created_unix_time asc").Find(&members)

	return members, err
}

// CanAccessBook returns whether the given user can access the given book
func (s *BookService) CanAccessBook(c core.Context, uid int64, bookId int64) (bool, error) {
	if uid <= 0 {
		return false, errs.ErrUserIdInvalid
	}

	if bookId <= 0 {
		return false, errs.ErrBookIdInvalid
	}

	if uid == bookId {
		return true, nil
	}

	book, err := s.GetSharedBookById(c, bookId)

	if err == errs.ErrSharedBookNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}

	if book.OwnerUid == uid {
		return true, nil
	}

	return s.UserDB().NewSession(c).Where("book_uid=? AND member_uid=?", bookId, uid).Exist(&models.BookMember{})
}

// AddBookMember adds a member to the given shared book
func (s *BookService) AddBookMember(c core.Context, book *models.SharedBook, memberUid int64) (*models.BookMember, error) {
	if memberUid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	if book.OwnerUid == memberUid {
		return nil, errs.ErrCannotAddYourselfAsBookMember
	}

	member := &models.BookMember{
		BookUid:         book.BookId,
		MemberUid:       memberUid,
		CreatedUnixTime: time.Now().Unix(),
	}

	err := s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		exists, err := sess.Where("book_uid=? AND member_uid=?", book.BookId, memberUid).Exist(&models.BookMember{})

		if err != nil {
			return err
		} else if exists {
			return errs.ErrBookMemberAlreadyExists
		}

		_, err = sess.Insert(member)
		return err
	})

	if err != nil {
		return nil, err
	}

	return member, nil
}

// RemoveBookMember removes a member from the given shared book, and stops sharing all accounts of the member to the book:
// an account which no transaction of the book uses is unshared, the sharing of the others is revoked, so the book keeps showing
// which account its existing transactions use, but cannot use the account for any new or changed transaction nor see its balance
func (s *BookService) RemoveBookMember(c core.Context, bookId int64, memberUid int64) error {
	if bookId <= 0 || memberUid <= 0 {
		return errs.ErrUserIdInvalid
	}

	err := s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		deletedRows, err := sess.Where("book_uid=? AND member_uid=?", bookId, memberUid).Delete(&models.BookMember{})

		if err != nil {
			return err
		} else if deletedRows < 1 {
			return errs.ErrBookMemberNotFound
		}

		updateModel := &models.SharedBookAccount{
			RevokedUnixTime: time.Now().Unix(),
		}

		_, err = sess.Cols("revoked_unix_time").Where("book_id=? AND owner_uid=? AND revoked_unix_time=?", bookId, memberUid, 0).Update(updateModel)
		return err
	})

	if err != nil {
		return err
	}

	var revokedSharedAccounts []*models.SharedBookAccount
	err = s.UserDB().NewSession(c).Where("book_id=? AND owner_uid=? AND revoked_unix_time>?", bookId, memberUid, 0).Find(&revokedSharedAccounts)

	if err != nil {
		log.Warnf(c, "[books.RemoveBookMember] failed to get revoked shared accounts of user \"uid:%d\" in book \"id:%d\", because %s", memberUid, bookId, err.Error())
		return nil
	}

	for i := 0; i < len(revokedSharedAccounts); i++ {
		err = s.deleteRevokedSharedAccountIfUnused(c, bookId, revokedSharedAccounts[i].AccountId)

		if err != nil {
			log.Warnf(c, "[books.RemoveBookMember] failed to delete unused revoked shared account \"id:%d\" in book \"id:%d\", because %s", revokedSharedAccounts[i].AccountId, bookId, err.Error())
		}
	}

	return nil
}

// GetSharedAccounts returns all accounts which are being shared to the given book, without the accounts whose sharing was revoked
func (s *BookService) GetSharedAccounts(c core.Context, bookId int64) ([]*models.SharedBookAccount, error) {
	if bookId <= 0 {
		return nil, errs.ErrBookIdInvalid
	}

	var sharedAccounts []*models.SharedBookAccount
	err := s.UserDB().NewSession(c).Where("book_id=? AND revoked_unix_time=?", bookId, 0).OrderBy("created_unix_time asc").Find(&sharedAccounts)

	return sharedAccounts, err
}

// GetSharedAccountIds returns the ids of all accounts shared to the given book
func (s *BookService) GetSharedAccountIds(c core.Context, bookId int64) ([]int64, error) {
	sharedAccounts, err := s.GetSharedAccounts(c, bookId)

	if err != nil {
		return nil, err
	}

	accountIds := make([]int64, len(sharedAccounts))

	for i := 0; i < len(sharedAccounts); i++ {
		accountIds[i] = sharedAccounts[i].AccountId
	}

	return accountIds, nil
}

// GetBookIdsSharingAccount returns the ids of all books which the given account is shared to
func (s *BookService) GetBookIdsSharingAccount(c core.Context, accountId int64) ([]int64, error) {
	var sharedAccounts []*models.SharedBookAccount
	err := s.UserDB().NewSession(c).Where("account_id=?", accountId).Find(&sharedAccounts)

	if err != nil {
		return nil, err
	}

	bookIds := make([]int64, len(sharedAccounts))

	for i := 0; i < len(sharedAccounts); i++ {
		bookIds[i] = sharedAccounts[i].BookId
	}

	return bookIds, nil
}

// IsAccountSharedToAnyBook returns whether the given account is shared to a book, or its sharing was revoked while transactions of the book still use it
func (s *BookService) IsAccountSharedToAnyBook(c core.Context, accountId int64) (bool, error) {
	return s.UserDB().NewSession(c).Where("account_id=?", accountId).Exist(&models.SharedBookAccount{})
}

// ShareAccount shares an account of the given user to the given shared book, or shares it again when its sharing was revoked
func (s *BookService) ShareAccount(c core.Context, bookId int64, ownerUid int64, accountId int64) (*models.SharedBookAccount, error) {
	if bookId <= 0 || ownerUid <= 0 || accountId <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	account := &models.Account{}
	has, err := s.UserDataDB(ownerUid).NewSession(c).ID(accountId).Where("uid=? AND deleted=?", ownerUid, false).Get(account)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, errs.ErrAccountNotFound
	}

	if account.ParentAccountId != models.LevelOneAccountParentId || account.Type != models.ACCOUNT_TYPE_SINGLE_ACCOUNT {
		return nil, errs.ErrAccountCannotBeShared
	}

	sharedAccount := &models.SharedBookAccount{
		BookId:          bookId,
		AccountId:       accountId,
		OwnerUid:        ownerUid,
		CreatedUnixTime: time.Now().Unix(),
	}

	err = s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		existingSharedAccount := &models.SharedBookAccount{}
		exists, err := sess.Where("book_id=? AND account_id=?", bookId, accountId).Get(existingSharedAccount)

		if err != nil {
			return err
		} else if exists && !existingSharedAccount.IsRevoked() {
			return errs.ErrAccountAlreadySharedToBook
		} else if exists {
			// the account was shared before and transactions of the book still use it, so make the existing sharing active again
			sharedAccount.CreatedUnixTime = existingSharedAccount.CreatedUnixTime
			sharedAccount.RevokedUnixTime = 0
			_, err = sess.Cols("owner_uid", "revoked_unix_time").Where("book_id=? AND account_id=?", bookId, accountId).Update(sharedAccount)
			return err
		}

		_, err = sess.Insert(sharedAccount)
		return err
	})

	if err != nil {
		return nil, err
	}

	return sharedAccount, nil
}

// UnshareAccount stops sharing an account to the given shared book, when transactions in the book still use the account,
// its sharing is revoked instead, so those transactions keep showing the account, but the book can no longer use it nor see its balance
func (s *BookService) UnshareAccount(c core.Context, bookId int64, accountId int64) error {
	if bookId <= 0 || accountId <= 0 {
		return errs.ErrBookIdInvalid
	}

	err := s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		updateModel := &models.SharedBookAccount{
			RevokedUnixTime: time.Now().Unix(),
		}

		updatedRows, err := sess.Cols("revoked_unix_time").Where("book_id=? AND account_id=? AND revoked_unix_time=?", bookId, accountId, 0).Update(updateModel)

		if err != nil {
			return err
		} else if updatedRows < 1 {
			return errs.ErrSharedAccountNotFound
		}

		return nil
	})

	if err != nil {
		return err
	}

	return s.deleteRevokedSharedAccountIfUnused(c, bookId, accountId)
}

// deleteRevokedSharedAccountIfUnused deletes the revoked sharing of an account to the given book when no transaction of the book uses the account
func (s *BookService) deleteRevokedSharedAccountIfUnused(c core.Context, bookId int64, accountId int64) error {
	inUse, err := s.UserDataDB(bookId).NewSession(c).Where("uid=? AND deleted=? AND (account_id=? OR related_account_id=?)", bookId, false, accountId, accountId).Exist(&models.Transaction{})

	if err != nil {
		return err
	} else if inUse {
		return nil
	}

	_, err = s.UserDB().NewSession(c).Where("book_id=? AND account_id=? AND revoked_unix_time>?", bookId, accountId, 0).Delete(&models.SharedBookAccount{})
	return err
}
