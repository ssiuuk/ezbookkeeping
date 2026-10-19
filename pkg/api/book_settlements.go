package api

import (
	"sort"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
)

// BookSettleUpHandler returns what each member of the current shared book paid, their fair share, what they owe or are owed, and the payments which settle it up,
// see services.BookSettlementService for how it is worked out
func (a *BooksApi) BookSettleUpHandler(c *core.WebContext) (any, *errs.Error) {
	var settleUpReq models.BookSettleUpRequest
	err := c.ShouldBindQuery(&settleUpReq)

	if err != nil {
		log.Warnf(c, "[book_settlements.BookSettleUpHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	book, errResp := a.getCurrentSharedBook(c)

	if errResp != nil {
		return nil, errResp
	}

	owner, err := a.users.GetUserById(c, book.OwnerUid)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[book_settlements.BookSettleUpHandler] failed to get book owner \"uid:%d\", because %s", book.OwnerUid, err.Error())
		}

		return nil, errs.ErrUserNotFound
	}

	members, err := a.books.GetBookMembers(c, book.BookId)

	if err != nil {
		log.Errorf(c, "[book_settlements.BookSettleUpHandler] failed to get book members for book \"id:%d\", because %s", book.BookId, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	paidAmountsByCurrency, err := a.settlements.GetSettleUpExpenses(c, book.BookId, settleUpReq.StartTime, settleUpReq.EndTime)

	if err != nil {
		log.Errorf(c, "[book_settlements.BookSettleUpHandler] failed to get expenses of book \"id:%d\", because %s", book.BookId, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	settlements, err := a.settlements.GetSettlements(c, book.BookId, settleUpReq.StartTime, settleUpReq.EndTime)

	if err != nil {
		log.Errorf(c, "[book_settlements.BookSettleUpHandler] failed to get settlements of book \"id:%d\", because %s", book.BookId, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	// the expenses are split among the current members and anyone else who paid any of them
	currentMemberUids := make(map[int64]bool, len(members)+1)
	splitUids := make([]int64, 0, len(members)+1)
	splitUids = append(splitUids, book.OwnerUid)
	currentMemberUids[book.OwnerUid] = true

	for i := 0; i < len(members); i++ {
		if !currentMemberUids[members[i].MemberUid] {
			splitUids = append(splitUids, members[i].MemberUid)
			currentMemberUids[members[i].MemberUid] = true
		}
	}

	formerPayerUids := make([]int64, 0)

	for _, paidAmounts := range paidAmountsByCurrency {
		for uid := range paidAmounts {
			if !currentMemberUids[uid] && !containsUid(formerPayerUids, uid) {
				formerPayerUids = append(formerPayerUids, uid)
			}
		}
	}

	sort.Slice(formerPayerUids, func(i, j int) bool {
		return formerPayerUids[i] < formerPayerUids[j]
	})

	splitUids = append(splitUids, formerPayerUids...)

	settlementsByCurrency := make(map[string][]*models.BookSettlement)

	for i := 0; i < len(settlements); i++ {
		settlementsByCurrency[settlements[i].Currency] = append(settlementsByCurrency[settlements[i].Currency], settlements[i])
	}

	currencies := make([]string, 0, len(paidAmountsByCurrency)+len(settlementsByCurrency))

	for currency := range paidAmountsByCurrency {
		currencies = append(currencies, currency)
	}

	for currency := range settlementsByCurrency {
		if _, exists := paidAmountsByCurrency[currency]; !exists {
			currencies = append(currencies, currency)
		}
	}

	// the default currency of the book owner comes first
	sort.Slice(currencies, func(i, j int) bool {
		if (currencies[i] == owner.DefaultCurrency) != (currencies[j] == owner.DefaultCurrency) {
			return currencies[i] == owner.DefaultCurrency
		}

		return currencies[i] < currencies[j]
	})

	resp := &models.BookSettleUpResponse{
		DefaultCurrency: owner.DefaultCurrency,
		StartTime:       settleUpReq.StartTime,
		EndTime:         settleUpReq.EndTime,
		Members:         make([]*models.BookSettleUpMemberResponse, 0, len(splitUids)),
		Currencies:      make([]*models.BookSettleUpCurrencyResponse, 0, len(currencies)),
	}

	memberUids := make([]int64, 0, len(splitUids))
	memberUids = append(memberUids, splitUids...)

	for i := 0; i < len(currencies); i++ {
		paidAmounts := paidAmountsByCurrency[currencies[i]]
		balances := services.CalculateSettleUpBalances(splitUids, paidAmounts, settlementsByCurrency[currencies[i]])
		totalExpense := int64(0)

		for _, amount := range paidAmounts {
			totalExpense += amount
		}

		for j := 0; j < len(balances); j++ {
			if !containsUid(memberUids, balances[j].Uid) {
				memberUids = append(memberUids, balances[j].Uid)
			}
		}

		resp.Currencies = append(resp.Currencies, &models.BookSettleUpCurrencyResponse{
			Currency:          currencies[i],
			TotalExpense:      totalExpense,
			Balances:          balances,
			SuggestedPayments: services.SuggestSettleUpPayments(balances),
		})
	}

	for i := 0; i < len(memberUids); i++ {
		memberResp := &models.BookSettleUpMemberResponse{
			Id:       memberUids[i],
			IsOwner:  memberUids[i] == book.OwnerUid,
			IsFormer: !currentMemberUids[memberUids[i]],
		}

		if memberUids[i] == owner.Uid {
			memberResp.Username = owner.Username
			memberResp.Nickname = owner.Nickname
		} else if user, err := a.users.GetUserById(c, memberUids[i]); err == nil {
			memberResp.Username = user.Username
			memberResp.Nickname = user.Nickname
		} else {
			log.Warnf(c, "[book_settlements.BookSettleUpHandler] cannot get member \"uid:%d\" of book \"id:%d\", because %s", memberUids[i], book.BookId, err.Error())
		}

		resp.Members = append(resp.Members, memberResp)
	}

	return resp, nil
}

// BookSettlementListHandler returns the settlements of the current shared book, the latest first
func (a *BooksApi) BookSettlementListHandler(c *core.WebContext) (any, *errs.Error) {
	var settlementListReq models.BookSettleUpRequest
	err := c.ShouldBindQuery(&settlementListReq)

	if err != nil {
		log.Warnf(c, "[book_settlements.BookSettlementListHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	book, errResp := a.getCurrentSharedBook(c)

	if errResp != nil {
		return nil, errResp
	}

	settlements, err := a.settlements.GetSettlements(c, book.BookId, settlementListReq.StartTime, settlementListReq.EndTime)

	if err != nil {
		log.Errorf(c, "[book_settlements.BookSettlementListHandler] failed to get settlements of book \"id:%d\", because %s", book.BookId, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	settlementResps := make([]*models.BookSettlementInfoResponse, len(settlements))

	for i := 0; i < len(settlements); i++ {
		settlementResps[i] = settlements[i].ToBookSettlementInfoResponse()
	}

	return settlementResps, nil
}

// BookSettlementCreateHandler records a settlement between two members of the current shared book, any member can record one
func (a *BooksApi) BookSettlementCreateHandler(c *core.WebContext) (any, *errs.Error) {
	var settlementCreateReq models.BookSettlementCreateRequest
	err := c.ShouldBindJSON(&settlementCreateReq)

	if err != nil {
		log.Warnf(c, "[book_settlements.BookSettlementCreateHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	book, errResp := a.getCurrentSharedBook(c)

	if errResp != nil {
		return nil, errResp
	}

	if settlementCreateReq.FromUid == settlementCreateReq.ToUid {
		return nil, errs.ErrBookSettlementSameMember
	}

	for _, memberUid := range []int64{settlementCreateReq.FromUid, settlementCreateReq.ToUid} {
		isParticipant, err := a.settlements.IsSettleUpParticipant(c, book, memberUid)

		if err != nil {
			log.Errorf(c, "[book_settlements.BookSettlementCreateHandler] failed to check whether user \"uid:%d\" is in book \"id:%d\", because %s", memberUid, book.BookId, err.Error())
			return nil, errs.Or(err, errs.ErrOperationFailed)
		} else if !isParticipant {
			return nil, errs.ErrBookSettlementMemberInvalid
		}
	}

	settlement := &models.BookSettlement{
		BookId:         book.BookId,
		FromUid:        settlementCreateReq.FromUid,
		ToUid:          settlementCreateReq.ToUid,
		Amount:         settlementCreateReq.Amount,
		Currency:       settlementCreateReq.Currency,
		SettlementTime: settlementCreateReq.Time,
		CreatedByUid:   uid,
		Comment:        settlementCreateReq.Comment,
	}

	err = a.settlements.CreateSettlement(c, settlement)

	if err != nil {
		log.Errorf(c, "[book_settlements.BookSettlementCreateHandler] failed to create settlement in book \"id:%d\" for user \"uid:%d\", because %s", book.BookId, uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[book_settlements.BookSettlementCreateHandler] user \"uid:%d\" has recorded settlement \"id:%d\" in book \"id:%d\"", uid, settlement.SettlementId, book.BookId)

	return settlement.ToBookSettlementInfoResponse(), nil
}

// BookSettlementDeleteHandler deletes a settlement of the current shared book,
// the member who recorded it, paid it or received it and the book owner can delete it
func (a *BooksApi) BookSettlementDeleteHandler(c *core.WebContext) (any, *errs.Error) {
	var settlementDeleteReq models.BookSettlementDeleteRequest
	err := c.ShouldBindJSON(&settlementDeleteReq)

	if err != nil {
		log.Warnf(c, "[book_settlements.BookSettlementDeleteHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	book, errResp := a.getCurrentSharedBook(c)

	if errResp != nil {
		return nil, errResp
	}

	settlement, err := a.settlements.GetSettlementById(c, book.BookId, settlementDeleteReq.Id)

	if err != nil {
		if !errs.IsCustomError(err) {
			log.Errorf(c, "[book_settlements.BookSettlementDeleteHandler] failed to get settlement \"id:%d\" of book \"id:%d\", because %s", settlementDeleteReq.Id, book.BookId, err.Error())
		}

		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	if book.OwnerUid != uid && settlement.CreatedByUid != uid && settlement.FromUid != uid && settlement.ToUid != uid {
		return nil, errs.ErrCannotDeleteBookSettlement
	}

	err = a.settlements.DeleteSettlement(c, book.BookId, settlement.SettlementId)

	if err != nil {
		log.Errorf(c, "[book_settlements.BookSettlementDeleteHandler] failed to delete settlement \"id:%d\" of book \"id:%d\", because %s", settlement.SettlementId, book.BookId, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	log.Infof(c, "[book_settlements.BookSettlementDeleteHandler] user \"uid:%d\" has deleted settlement \"id:%d\" in book \"id:%d\"", uid, settlement.SettlementId, book.BookId)

	return true, nil
}

// getCurrentSharedBook returns the current book when it is a shared book, current user can access it as the book is chosen by the request
func (a *BooksApi) getCurrentSharedBook(c *core.WebContext) (*models.SharedBook, *errs.Error) {
	bookId := c.GetCurrentBookUid()

	if bookId == c.GetCurrentUid() {
		return nil, errs.ErrNotInSharedBook
	}

	book, err := a.books.GetSharedBookById(c, bookId)

	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return book, nil
}

func containsUid(uids []int64, uid int64) bool {
	for i := 0; i < len(uids); i++ {
		if uids[i] == uid {
			return true
		}
	}

	return false
}
