package services

import (
	"time"

	"xorm.io/xorm"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
)

// BookService represents shared book service
type BookService struct {
	ServiceUsingDB
}

// Initialize a shared book service singleton instance
var (
	Books = &BookService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
	}
)

// GetBookMembers returns all members of the book owned by the given user
func (s *BookService) GetBookMembers(c core.Context, bookUid int64) ([]*models.BookMember, error) {
	if bookUid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var members []*models.BookMember
	err := s.UserDB().NewSession(c).Where("book_uid=?", bookUid).OrderBy("created_unix_time asc").Find(&members)

	return members, err
}

// GetSharedBooks returns all books shared with the given user
func (s *BookService) GetSharedBooks(c core.Context, memberUid int64) ([]*models.BookMember, error) {
	if memberUid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	var members []*models.BookMember
	err := s.UserDB().NewSession(c).Where("member_uid=?", memberUid).OrderBy("created_unix_time asc").Find(&members)

	return members, err
}

// CanAccessBook returns whether the given user can access the book of the given owner
func (s *BookService) CanAccessBook(c core.Context, uid int64, bookUid int64) (bool, error) {
	if uid <= 0 {
		return false, errs.ErrUserIdInvalid
	}

	if bookUid <= 0 {
		return false, errs.ErrBookIdInvalid
	}

	if uid == bookUid {
		return true, nil
	}

	return s.UserDB().NewSession(c).Where("book_uid=? AND member_uid=?", bookUid, uid).Exist(&models.BookMember{})
}

// AddBookMember adds a member to the book owned by the given user
func (s *BookService) AddBookMember(c core.Context, bookUid int64, memberUid int64) (*models.BookMember, error) {
	if bookUid <= 0 || memberUid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	if bookUid == memberUid {
		return nil, errs.ErrCannotAddYourselfAsBookMember
	}

	member := &models.BookMember{
		BookUid:         bookUid,
		MemberUid:       memberUid,
		CreatedUnixTime: time.Now().Unix(),
	}

	err := s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		exists, err := sess.Where("book_uid=? AND member_uid=?", bookUid, memberUid).Exist(&models.BookMember{})

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

// RemoveBookMember removes a member from the book owned by the given user
func (s *BookService) RemoveBookMember(c core.Context, bookUid int64, memberUid int64) error {
	if bookUid <= 0 || memberUid <= 0 {
		return errs.ErrUserIdInvalid
	}

	return s.UserDB().DoTransaction(c, func(sess *xorm.Session) error {
		deletedRows, err := sess.Where("book_uid=? AND member_uid=?", bookUid, memberUid).Delete(&models.BookMember{})

		if err != nil {
			return err
		} else if deletedRows < 1 {
			return errs.ErrBookMemberNotFound
		}

		return nil
	})
}
