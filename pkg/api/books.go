package api

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
)

// BooksApi represents shared book api
type BooksApi struct {
	users *services.UserService
	books *services.BookService
}

// Initialize a shared book api singleton instance
var (
	Books = &BooksApi{
		users: services.Users,
		books: services.Books,
	}
)

// BookListHandler returns the books which current user can access, the first one is the user's own book
func (a *BooksApi) BookListHandler(c *core.WebContext) (any, *errs.Error) {
	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[books.BookListHandler] failed to get user \"uid:%d\", because %s", uid, err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	sharedBooks, err := a.books.GetSharedBooks(c, uid)

	if err != nil {
		log.Errorf(c, "[books.BookListHandler] failed to get shared books for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	bookResps := make([]*models.BookInfoResponse, 0, len(sharedBooks)+1)
	bookResps = append(bookResps, &models.BookInfoResponse{
		Id:            user.Uid,
		OwnerUsername: user.Username,
		OwnerNickname: user.Nickname,
		IsOwner:       true,
	})

	for i := 0; i < len(sharedBooks); i++ {
		owner, err := a.users.GetUserById(c, sharedBooks[i].BookUid)

		if err != nil || owner.Disabled {
			continue
		}

		bookResps = append(bookResps, &models.BookInfoResponse{
			Id:            owner.Uid,
			OwnerUsername: owner.Username,
			OwnerNickname: owner.Nickname,
			IsOwner:       false,
		})
	}

	return bookResps, nil
}

// BookMemberListHandler returns the owner and members of the current book
func (a *BooksApi) BookMemberListHandler(c *core.WebContext) (any, *errs.Error) {
	bookUid := c.GetCurrentBookUid()
	owner, err := a.users.GetUserById(c, bookUid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[books.BookMemberListHandler] failed to get book owner \"uid:%d\", because %s", bookUid, err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	members, err := a.books.GetBookMembers(c, bookUid)

	if err != nil {
		log.Errorf(c, "[books.BookMemberListHandler] failed to get book members for book \"uid:%d\", because %s", bookUid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	memberResps := make([]*models.BookMemberInfoResponse, 0, len(members)+1)
	memberResps = append(memberResps, &models.BookMemberInfoResponse{
		Id:       owner.Uid,
		Username: owner.Username,
		Nickname: owner.Nickname,
		IsOwner:  true,
	})

	for i := 0; i < len(members); i++ {
		member, err := a.users.GetUserById(c, members[i].MemberUid)

		if err != nil {
			continue
		}

		memberResps = append(memberResps, &models.BookMemberInfoResponse{
			Id:        member.Uid,
			Username:  member.Username,
			Nickname:  member.Nickname,
			CreatedAt: members[i].CreatedUnixTime,
		})
	}

	return memberResps, nil
}

// BookMemberAddHandler adds a member to current user's own book
func (a *BooksApi) BookMemberAddHandler(c *core.WebContext) (any, *errs.Error) {
	var memberAddReq models.BookMemberAddRequest
	err := c.ShouldBindJSON(&memberAddReq)

	if err != nil {
		log.Warnf(c, "[books.BookMemberAddHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()

	if c.GetCurrentBookUid() != uid {
		return nil, errs.ErrBookMemberCannotManageMembers
	}

	member, err := a.users.GetUserByUsername(c, memberAddReq.Username)

	if err != nil || member.Disabled {
		return nil, errs.ErrUserNotFound
	}

	bookMember, err := a.books.AddBookMember(c, uid, member.Uid)

	if err != nil {
		log.Errorf(c, "[books.BookMemberAddHandler] failed to add user \"uid:%d\" to book of user \"uid:%d\", because %s", member.Uid, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[books.BookMemberAddHandler] user \"uid:%d\" has added user \"uid:%d\" to book", uid, member.Uid)

	return &models.BookMemberInfoResponse{
		Id:        member.Uid,
		Username:  member.Username,
		Nickname:  member.Nickname,
		CreatedAt: bookMember.CreatedUnixTime,
	}, nil
}

// BookMemberRemoveHandler removes a member from current user's own book
func (a *BooksApi) BookMemberRemoveHandler(c *core.WebContext) (any, *errs.Error) {
	var memberRemoveReq models.BookMemberRemoveRequest
	err := c.ShouldBindJSON(&memberRemoveReq)

	if err != nil {
		log.Warnf(c, "[books.BookMemberRemoveHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()

	if c.GetCurrentBookUid() != uid {
		return nil, errs.ErrBookMemberCannotManageMembers
	}

	err = a.books.RemoveBookMember(c, uid, memberRemoveReq.MemberId)

	if err != nil {
		log.Errorf(c, "[books.BookMemberRemoveHandler] failed to remove user \"uid:%d\" from book of user \"uid:%d\", because %s", memberRemoveReq.MemberId, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[books.BookMemberRemoveHandler] user \"uid:%d\" has removed user \"uid:%d\" from book", uid, memberRemoveReq.MemberId)

	return true, nil
}

// BookLeaveHandler removes current user from a book shared with them
func (a *BooksApi) BookLeaveHandler(c *core.WebContext) (any, *errs.Error) {
	var leaveReq models.BookLeaveRequest
	err := c.ShouldBindJSON(&leaveReq)

	if err != nil {
		log.Warnf(c, "[books.BookLeaveHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	err = a.books.RemoveBookMember(c, leaveReq.BookId, uid)

	if err != nil {
		log.Errorf(c, "[books.BookLeaveHandler] failed to remove user \"uid:%d\" from book of user \"uid:%d\", because %s", uid, leaveReq.BookId, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[books.BookLeaveHandler] user \"uid:%d\" has left book of user \"uid:%d\"", uid, leaveReq.BookId)

	return true, nil
}
