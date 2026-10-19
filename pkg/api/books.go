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
	users       *services.UserService
	books       *services.BookService
	settlements *services.BookSettlementService
}

// Initialize a shared book api singleton instance
var (
	Books = &BooksApi{
		users:       services.Users,
		books:       services.Books,
		settlements: services.BookSettlements,
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

	sharedBooks, err := a.books.GetAccessibleSharedBooks(c, uid)

	if err != nil {
		log.Errorf(c, "[books.BookListHandler] failed to get shared books for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	bookResps := make([]*models.BookInfoResponse, 0, len(sharedBooks)+1)
	bookResps = append(bookResps, &models.BookInfoResponse{
		Id:            user.Uid,
		OwnerUsername: user.Username,
		OwnerNickname: user.Nickname,
		IsPersonal:    true,
		IsOwner:       true,
	})

	for i := 0; i < len(sharedBooks); i++ {
		owner := user

		if sharedBooks[i].OwnerUid != uid {
			owner, err = a.users.GetUserById(c, sharedBooks[i].OwnerUid)

			if err != nil {
				continue
			}
		}

		bookResps = append(bookResps, a.getBookInfoResponse(sharedBooks[i], owner, uid))
	}

	return bookResps, nil
}

// BookCreateHandler creates a new shared book owned by current user
func (a *BooksApi) BookCreateHandler(c *core.WebContext) (any, *errs.Error) {
	var bookCreateReq models.BookCreateRequest
	err := c.ShouldBindJSON(&bookCreateReq)

	if err != nil {
		log.Warnf(c, "[books.BookCreateHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	user, err := a.users.GetUserById(c, uid)

	if err != nil {
		return nil, errs.ErrUserNotFound
	}

	book, err := a.books.CreateSharedBook(c, uid, bookCreateReq.Name)

	if err != nil {
		log.Errorf(c, "[books.BookCreateHandler] failed to create shared book for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[books.BookCreateHandler] user \"uid:%d\" has created shared book \"id:%d\"", uid, book.BookId)

	return a.getBookInfoResponse(book, user, uid), nil
}

// BookModifyHandler renames a shared book owned by current user
func (a *BooksApi) BookModifyHandler(c *core.WebContext) (any, *errs.Error) {
	var bookModifyReq models.BookModifyRequest
	err := c.ShouldBindJSON(&bookModifyReq)

	if err != nil {
		log.Warnf(c, "[books.BookModifyHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	err = a.books.ModifySharedBookName(c, uid, bookModifyReq.Id, bookModifyReq.Name)

	if err != nil {
		log.Errorf(c, "[books.BookModifyHandler] failed to modify shared book \"id:%d\" for user \"uid:%d\", because %s", bookModifyReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return true, nil
}

// BookDeleteHandler deletes an empty shared book owned by current user
func (a *BooksApi) BookDeleteHandler(c *core.WebContext) (any, *errs.Error) {
	var bookDeleteReq models.BookDeleteRequest
	err := c.ShouldBindJSON(&bookDeleteReq)

	if err != nil {
		log.Warnf(c, "[books.BookDeleteHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	err = a.books.DeleteSharedBook(c, uid, bookDeleteReq.Id)

	if err != nil {
		log.Errorf(c, "[books.BookDeleteHandler] failed to delete shared book \"id:%d\" for user \"uid:%d\", because %s", bookDeleteReq.Id, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	err = a.settlements.DeleteAllSettlementsOfBook(c, bookDeleteReq.Id)

	if err != nil {
		log.Warnf(c, "[books.BookDeleteHandler] failed to delete settlements of shared book \"id:%d\", because %s", bookDeleteReq.Id, err.Error())
	}

	log.Infof(c, "[books.BookDeleteHandler] user \"uid:%d\" has deleted shared book \"id:%d\"", uid, bookDeleteReq.Id)

	return true, nil
}

// BookMemberListHandler returns the owner and members of the current book
func (a *BooksApi) BookMemberListHandler(c *core.WebContext) (any, *errs.Error) {
	uid := c.GetCurrentUid()
	bookId := c.GetCurrentBookUid()

	if bookId == uid {
		user, err := a.users.GetUserById(c, uid)

		if err != nil {
			return nil, errs.ErrUserNotFound
		}

		return []*models.BookMemberInfoResponse{
			{
				Id:       user.Uid,
				Username: user.Username,
				Nickname: user.Nickname,
				IsOwner:  true,
			},
		}, nil
	}

	book, err := a.books.GetSharedBookById(c, bookId)

	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	owner, err := a.users.GetUserById(c, book.OwnerUid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[books.BookMemberListHandler] failed to get book owner \"uid:%d\", because %s", book.OwnerUid, err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	members, err := a.books.GetBookMembers(c, bookId)

	if err != nil {
		log.Errorf(c, "[books.BookMemberListHandler] failed to get book members for book \"id:%d\", because %s", bookId, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	memberResps := make([]*models.BookMemberInfoResponse, 0, len(members)+1)
	memberResps = append(memberResps, &models.BookMemberInfoResponse{
		Id:        owner.Uid,
		Username:  owner.Username,
		Nickname:  owner.Nickname,
		IsOwner:   true,
		CreatedAt: book.CreatedUnixTime,
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

// BookMemberAddHandler adds a member to the current shared book
func (a *BooksApi) BookMemberAddHandler(c *core.WebContext) (any, *errs.Error) {
	var memberAddReq models.BookMemberAddRequest
	err := c.ShouldBindJSON(&memberAddReq)

	if err != nil {
		log.Warnf(c, "[books.BookMemberAddHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	book, errResp := a.getCurrentSharedBookOwnedByCurrentUser(c)

	if errResp != nil {
		return nil, errResp
	}

	member, err := a.users.GetUserByUsername(c, memberAddReq.Username)

	if err != nil || member.Disabled {
		return nil, errs.ErrUserNotFound
	}

	bookMember, err := a.books.AddBookMember(c, book, member.Uid)

	if err != nil {
		log.Errorf(c, "[books.BookMemberAddHandler] failed to add user \"uid:%d\" to book \"id:%d\", because %s", member.Uid, book.BookId, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[books.BookMemberAddHandler] user \"uid:%d\" has added user \"uid:%d\" to book \"id:%d\"", uid, member.Uid, book.BookId)

	return &models.BookMemberInfoResponse{
		Id:        member.Uid,
		Username:  member.Username,
		Nickname:  member.Nickname,
		CreatedAt: bookMember.CreatedUnixTime,
	}, nil
}

// BookMemberRemoveHandler removes a member from the current shared book
func (a *BooksApi) BookMemberRemoveHandler(c *core.WebContext) (any, *errs.Error) {
	var memberRemoveReq models.BookMemberRemoveRequest
	err := c.ShouldBindJSON(&memberRemoveReq)

	if err != nil {
		log.Warnf(c, "[books.BookMemberRemoveHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	book, errResp := a.getCurrentSharedBookOwnedByCurrentUser(c)

	if errResp != nil {
		return nil, errResp
	}

	err = a.books.RemoveBookMember(c, book.BookId, memberRemoveReq.MemberId)

	if err != nil {
		log.Errorf(c, "[books.BookMemberRemoveHandler] failed to remove user \"uid:%d\" from book \"id:%d\", because %s", memberRemoveReq.MemberId, book.BookId, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[books.BookMemberRemoveHandler] user \"uid:%d\" has removed user \"uid:%d\" from book \"id:%d\"", uid, memberRemoveReq.MemberId, book.BookId)

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
		log.Errorf(c, "[books.BookLeaveHandler] failed to remove user \"uid:%d\" from book \"id:%d\", because %s", uid, leaveReq.BookId, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[books.BookLeaveHandler] user \"uid:%d\" has left book \"id:%d\"", uid, leaveReq.BookId)

	return true, nil
}

// BookSharedAccountListHandler returns the accounts shared to the current shared book
func (a *BooksApi) BookSharedAccountListHandler(c *core.WebContext) (any, *errs.Error) {
	bookId := c.GetCurrentBookUid()

	if bookId == c.GetCurrentUid() {
		return []*models.BookSharedAccountInfoResponse{}, nil
	}

	sharedAccounts, err := a.books.GetSharedAccounts(c, bookId)

	if err != nil {
		log.Errorf(c, "[books.BookSharedAccountListHandler] failed to get shared accounts of book \"id:%d\", because %s", bookId, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	owners := make(map[int64]*models.User)
	resps := make([]*models.BookSharedAccountInfoResponse, 0, len(sharedAccounts))

	for i := 0; i < len(sharedAccounts); i++ {
		owner, exists := owners[sharedAccounts[i].OwnerUid]

		if !exists {
			owner, err = a.users.GetUserById(c, sharedAccounts[i].OwnerUid)

			if err != nil {
				continue
			}

			owners[owner.Uid] = owner
		}

		resps = append(resps, &models.BookSharedAccountInfoResponse{
			AccountId:     sharedAccounts[i].AccountId,
			OwnerUid:      owner.Uid,
			OwnerUsername: owner.Username,
			OwnerNickname: owner.Nickname,
		})
	}

	return resps, nil
}

// BookShareAccountHandler shares an account of current user to the current shared book
func (a *BooksApi) BookShareAccountHandler(c *core.WebContext) (any, *errs.Error) {
	var shareReq models.BookAccountShareRequest
	err := c.ShouldBindJSON(&shareReq)

	if err != nil {
		log.Warnf(c, "[books.BookShareAccountHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	bookId := c.GetCurrentBookUid()

	if bookId == uid {
		return nil, errs.ErrNotInSharedBook
	}

	_, err = a.books.ShareAccount(c, bookId, uid, shareReq.AccountId)

	if err != nil {
		log.Errorf(c, "[books.BookShareAccountHandler] failed to share account \"id:%d\" of user \"uid:%d\" to book \"id:%d\", because %s", shareReq.AccountId, uid, bookId, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[books.BookShareAccountHandler] user \"uid:%d\" has shared account \"id:%d\" to book \"id:%d\"", uid, shareReq.AccountId, bookId)

	return true, nil
}

// BookUnshareAccountHandler stops sharing an account to the current shared book
func (a *BooksApi) BookUnshareAccountHandler(c *core.WebContext) (any, *errs.Error) {
	var shareReq models.BookAccountShareRequest
	err := c.ShouldBindJSON(&shareReq)

	if err != nil {
		log.Warnf(c, "[books.BookUnshareAccountHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	bookId := c.GetCurrentBookUid()

	if bookId == uid {
		return nil, errs.ErrNotInSharedBook
	}

	book, err := a.books.GetSharedBookById(c, bookId)

	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	sharedAccounts, err := a.books.GetSharedAccounts(c, bookId)

	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	var sharedAccount *models.SharedBookAccount

	for i := 0; i < len(sharedAccounts); i++ {
		if sharedAccounts[i].AccountId == shareReq.AccountId {
			sharedAccount = sharedAccounts[i]
			break
		}
	}

	if sharedAccount == nil {
		return nil, errs.ErrSharedAccountNotFound
	}

	if sharedAccount.OwnerUid != uid && book.OwnerUid != uid {
		return nil, errs.ErrCannotManageSharedAccount
	}

	err = a.books.UnshareAccount(c, bookId, shareReq.AccountId)

	if err != nil {
		log.Errorf(c, "[books.BookUnshareAccountHandler] failed to stop sharing account \"id:%d\" to book \"id:%d\", because %s", shareReq.AccountId, bookId, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[books.BookUnshareAccountHandler] user \"uid:%d\" has stopped sharing account \"id:%d\" to book \"id:%d\"", uid, shareReq.AccountId, bookId)

	return true, nil
}

func (a *BooksApi) getCurrentSharedBookOwnedByCurrentUser(c *core.WebContext) (*models.SharedBook, *errs.Error) {
	uid := c.GetCurrentUid()
	bookId := c.GetCurrentBookUid()

	if bookId == uid {
		return nil, errs.ErrNotInSharedBook
	}

	book, err := a.books.GetSharedBookById(c, bookId)

	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if book.OwnerUid != uid {
		return nil, errs.ErrBookMemberCannotManageMembers
	}

	return book, nil
}

func (a *BooksApi) getBookInfoResponse(book *models.SharedBook, owner *models.User, currentUid int64) *models.BookInfoResponse {
	return &models.BookInfoResponse{
		Id:            book.BookId,
		Name:          book.Name,
		OwnerUsername: owner.Username,
		OwnerNickname: owner.Nickname,
		IsPersonal:    false,
		IsOwner:       book.OwnerUid == currentUid,
	}
}
