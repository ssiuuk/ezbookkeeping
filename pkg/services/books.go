package services

import (
	"time"

	"xorm.io/xorm"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
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

// GetBookOwnerUid returns the uid of the user who owns the given book,
// it is the owner of the shared book when the book is a shared book, otherwise the book is a personal book and owned by the user of the same uid
func (s *BookService) GetBookOwnerUid(c core.Context, bookUid int64) (int64, error) {
	if bookUid <= 0 {
		return 0, errs.ErrBookIdInvalid
	}

	book := &models.SharedBook{}
	has, err := s.UserDB().NewSession(c).ID(bookUid).Get(book)

	if err != nil {
		return 0, err
	} else if has {
		return book.OwnerUid, nil
	}

	return bookUid, nil
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

// RemoveBookMember removes a member from the given shared book
func (s *BookService) RemoveBookMember(c core.Context, bookId int64, memberUid int64) error {
	if bookId <= 0 || memberUid <= 0 {
		return errs.ErrUserIdInvalid
	}

	return s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		deletedRows, err := sess.Where("book_uid=? AND member_uid=?", bookId, memberUid).Delete(&models.BookMember{})

		if err != nil {
			return err
		} else if deletedRows < 1 {
			return errs.ErrBookMemberNotFound
		}

		return nil
	})
}

// GetSharedAccounts returns all accounts shared to the given book
func (s *BookService) GetSharedAccounts(c core.Context, bookId int64) ([]*models.SharedBookAccount, error) {
	if bookId <= 0 {
		return nil, errs.ErrBookIdInvalid
	}

	var sharedAccounts []*models.SharedBookAccount
	err := s.UserDB().NewSession(c).Where("book_id=?", bookId).OrderBy("created_unix_time asc").Find(&sharedAccounts)

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

// IsAccountSharedToAnyBook returns whether the given account or any of its sub-accounts is shared to a book
func (s *BookService) IsAccountSharedToAnyBook(c core.Context, accountId int64) (bool, error) {
	return s.UserDB().NewSession(c).Where("account_id=?", accountId).Exist(&models.SharedBookAccount{})
}

// ShareAccount shares an account of the given user to the given shared book
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
		exists, err := sess.Where("book_id=? AND account_id=?", bookId, accountId).Exist(&models.SharedBookAccount{})

		if err != nil {
			return err
		} else if exists {
			return errs.ErrAccountAlreadySharedToBook
		}

		_, err = sess.Insert(sharedAccount)
		return err
	})

	if err != nil {
		return nil, err
	}

	return sharedAccount, nil
}

// UnshareAccount stops sharing an account to the given shared book, it fails when any transaction in the book uses the account
func (s *BookService) UnshareAccount(c core.Context, bookId int64, accountId int64) error {
	if bookId <= 0 || accountId <= 0 {
		return errs.ErrBookIdInvalid
	}

	inUse, err := s.UserDataDB(bookId).NewSession(c).Where("uid=? AND deleted=? AND (account_id=? OR related_account_id=?)", bookId, false, accountId, accountId).Exist(&models.Transaction{})

	if err != nil {
		return err
	} else if inUse {
		return errs.ErrSharedAccountInUse
	}

	return s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		deletedRows, err := sess.Where("book_id=? AND account_id=?", bookId, accountId).Delete(&models.SharedBookAccount{})

		if err != nil {
			return err
		} else if deletedRows < 1 {
			return errs.ErrSharedAccountNotFound
		}

		return nil
	})
}
