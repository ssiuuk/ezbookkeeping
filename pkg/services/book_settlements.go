package services

import (
	"sort"
	"time"

	"xorm.io/xorm"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
)

// BookSettlementService represents the settle up and settlement service of shared books
//
// The settle up of a shared book works like this:
//   - an expense is paid by the member who owns the account it was paid from, an expense paid from an account of the book itself is paid by nobody and left out
//   - the expenses are split equally among the current members of the book and anyone else who paid any of them (e.g. a member who has left the book)
//   - income, transfers and balance modifications are left out, hidden amounts are counted like any other amount
//   - a settlement recorded from one member to another offsets what they owe each other
//   - amounts are never converted between currencies, each currency is settled up on its own
//
// Without a time range, all the expenses and settlements of the book are counted, so the balances show what is still owed now.
type BookSettlementService struct {
	ServiceUsingDB
	ServiceUsingUuid
}

// Initialize a book settlement service singleton instance
var (
	BookSettlements = &BookSettlementService{
		ServiceUsingDB: ServiceUsingDB{
			container: datastore.Container,
		},
		ServiceUsingUuid: ServiceUsingUuid{
			container: uuid.Container,
		},
	}
)

// GetSettleUpExpenses returns the total expense of the given book which each member paid, keyed by currency and then by the uid of the member,
// the expenses paid from the accounts of the book itself are left out, the times are unix times in seconds and 0 means not limited
func (s *BookSettlementService) GetSettleUpExpenses(c core.Context, bookId int64, startTime int64, endTime int64) (map[string]map[int64]int64, error) {
	if bookId <= 0 {
		return nil, errs.ErrBookIdInvalid
	}

	condition := "uid=? AND deleted=? AND type=?"
	conditionParams := []any{bookId, false, models.TRANSACTION_DB_TYPE_EXPENSE}

	if startTime > 0 {
		condition = condition + " AND transaction_time>=?"
		conditionParams = append(conditionParams, utils.GetMinTransactionTimeFromUnixTime(startTime))
	}

	if endTime > 0 {
		condition = condition + " AND transaction_time<=?"
		conditionParams = append(conditionParams, utils.GetMaxTransactionTimeFromUnixTime(endTime))
	}

	var accountExpenses []*models.BookSettleUpAccountExpense
	err := s.UserDataDB(bookId).NewSession(c).Table(&models.Transaction{}).
		Select("account_id, SUM(amount) AS total_amount").
		Where(condition, conditionParams...).
		GroupBy("account_id").
		Find(&accountExpenses)

	if err != nil {
		return nil, err
	}

	result := make(map[string]map[int64]int64)

	if len(accountExpenses) < 1 {
		return result, nil
	}

	accountIds := make([]int64, len(accountExpenses))

	for i := 0; i < len(accountExpenses); i++ {
		accountIds[i] = accountExpenses[i].AccountId
	}

	// the accounts of members who have left the book are still the accounts which paid the expenses, so they are read whatever their sharing is
	var accounts []*models.Account
	err = s.UserDataDB(bookId).NewSession(c).In("account_id", accountIds).Find(&accounts)

	if err != nil {
		return nil, err
	}

	accountMap := make(map[int64]*models.Account, len(accounts))

	for i := 0; i < len(accounts); i++ {
		accountMap[accounts[i].AccountId] = accounts[i]
	}

	for i := 0; i < len(accountExpenses); i++ {
		account, exists := accountMap[accountExpenses[i].AccountId]

		if !exists || account.Uid == bookId {
			continue
		}

		paidAmounts, exists := result[account.Currency]

		if !exists {
			paidAmounts = make(map[int64]int64)
			result[account.Currency] = paidAmounts
		}

		paidAmounts[account.Uid] += accountExpenses[i].TotalAmount
	}

	return result, nil
}

// GetSettlements returns the settlements of the given book, the latest first, the times are unix times in seconds and 0 means not limited
func (s *BookSettlementService) GetSettlements(c core.Context, bookId int64, startTime int64, endTime int64) ([]*models.BookSettlement, error) {
	if bookId <= 0 {
		return nil, errs.ErrBookIdInvalid
	}

	sess := s.UserDataDB(bookId).NewSession(c).Where("book_id=? AND deleted=?", bookId, false)

	if startTime > 0 {
		sess = sess.And("settlement_time>=?", startTime)
	}

	if endTime > 0 {
		sess = sess.And("settlement_time<=?", endTime)
	}

	var settlements []*models.BookSettlement
	err := sess.OrderBy("settlement_time desc, created_unix_time desc").Find(&settlements)

	return settlements, err
}

// GetSettlementById returns the settlement of the given id in the given book
func (s *BookSettlementService) GetSettlementById(c core.Context, bookId int64, settlementId int64) (*models.BookSettlement, error) {
	if bookId <= 0 {
		return nil, errs.ErrBookIdInvalid
	}

	if settlementId <= 0 {
		return nil, errs.ErrBookSettlementNotFound
	}

	settlement := &models.BookSettlement{}
	has, err := s.UserDataDB(bookId).NewSession(c).ID(settlementId).Where("book_id=? AND deleted=?", bookId, false).Get(settlement)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, errs.ErrBookSettlementNotFound
	}

	return settlement, nil
}

// IsSettleUpParticipant returns whether the given user can pay or receive a settlement in the given shared book,
// that is the owner, a member, or a former member whose account is still used by transactions of the book
func (s *BookSettlementService) IsSettleUpParticipant(c core.Context, book *models.SharedBook, uid int64) (bool, error) {
	if uid <= 0 {
		return false, nil
	}

	if book.OwnerUid == uid {
		return true, nil
	}

	isMember, err := s.UserDB().NewSession(c).Where("book_uid=? AND member_uid=?", book.BookId, uid).Exist(&models.BookMember{})

	if err != nil || isMember {
		return isMember, err
	}

	return s.UserDB().NewSession(c).Where("book_id=? AND owner_uid=?", book.BookId, uid).Exist(&models.SharedBookAccount{})
}

// CreateSettlement saves a new settlement
func (s *BookSettlementService) CreateSettlement(c core.Context, settlement *models.BookSettlement) error {
	if settlement.BookId <= 0 {
		return errs.ErrBookIdInvalid
	}

	if settlement.FromUid == settlement.ToUid {
		return errs.ErrBookSettlementSameMember
	}

	settlement.SettlementId = s.GenerateUuid(uuid.UUID_TYPE_SETTLEMENT)

	if settlement.SettlementId < 1 {
		return errs.ErrSystemIsBusy
	}

	now := time.Now().Unix()
	settlement.Deleted = false
	settlement.CreatedUnixTime = now
	settlement.UpdatedUnixTime = now

	if settlement.SettlementTime <= 0 {
		settlement.SettlementTime = now
	}

	return s.UserDataDB(settlement.BookId).DoTransaction(c, func(sess *xorm.Session) error {
		_, err := sess.Insert(settlement)
		return err
	})
}

// DeleteSettlement deletes a settlement of the given book
func (s *BookSettlementService) DeleteSettlement(c core.Context, bookId int64, settlementId int64) error {
	if bookId <= 0 {
		return errs.ErrBookIdInvalid
	}

	now := time.Now().Unix()

	updateModel := &models.BookSettlement{
		Deleted:         true,
		DeletedUnixTime: now,
	}

	return s.UserDataDB(bookId).DoTransaction(c, func(sess *xorm.Session) error {
		deletedRows, err := sess.ID(settlementId).Cols("deleted", "deleted_unix_time").Where("book_id=? AND deleted=?", bookId, false).Update(updateModel)

		if err != nil {
			return err
		} else if deletedRows < 1 {
			return errs.ErrBookSettlementNotFound
		}

		return nil
	})
}

// DeleteAllSettlementsOfBook deletes all settlements of the given book
func (s *BookSettlementService) DeleteAllSettlementsOfBook(c core.Context, bookId int64) error {
	if bookId <= 0 {
		return errs.ErrBookIdInvalid
	}

	updateModel := &models.BookSettlement{
		Deleted:         true,
		DeletedUnixTime: time.Now().Unix(),
	}

	return s.UserDataDB(bookId).DoTransaction(c, func(sess *xorm.Session) error {
		_, err := sess.Cols("deleted", "deleted_unix_time").Where("book_id=? AND deleted=?", bookId, false).Update(updateModel)
		return err
	})
}

// CalculateSettleUpBalances returns the balance of each member in one currency.
// The total of paidAmounts (keyed by the uid of the member who paid) is split equally among splitUids and anyone in paidAmounts,
// the cents which cannot be split equally go one each to the first members in that order, so the fair shares always add up to the total.
// Each settlement (all in the same currency) raises the balance of the member who paid it and lowers the balance of the member who received it.
// The balances are in the order of splitUids, then the other members who paid, then the members who only took part in settlements, they always add up to 0.
func CalculateSettleUpBalances(splitUids []int64, paidAmounts map[int64]int64, settlements []*models.BookSettlement) []*models.BookSettleUpMemberBalance {
	balances := make([]*models.BookSettleUpMemberBalance, 0, len(splitUids)+len(paidAmounts))
	balanceMap := make(map[int64]*models.BookSettleUpMemberBalance, len(splitUids)+len(paidAmounts))

	addMember := func(uid int64) {
		if _, exists := balanceMap[uid]; exists {
			return
		}

		balance := &models.BookSettleUpMemberBalance{Uid: uid}
		balances = append(balances, balance)
		balanceMap[uid] = balance
	}

	for i := 0; i < len(splitUids); i++ {
		addMember(splitUids[i])
	}

	for _, uid := range getSortedUids(paidAmounts) {
		addMember(uid)
	}

	splitCount := int64(len(balances))
	totalPaid := int64(0)

	for uid, amount := range paidAmounts {
		balanceMap[uid].Paid += amount
		totalPaid += amount
	}

	if splitCount > 0 {
		fairShare := totalPaid / splitCount
		remainder := totalPaid % splitCount

		for i := int64(0); i < splitCount; i++ {
			balances[i].FairShare = fairShare

			if i < remainder {
				balances[i].FairShare++
			} else if remainder < 0 && i < -remainder {
				balances[i].FairShare--
			}
		}
	}

	settlementUids := make(map[int64]int64)

	for i := 0; i < len(settlements); i++ {
		settlementUids[settlements[i].FromUid] = 0
		settlementUids[settlements[i].ToUid] = 0
	}

	for _, uid := range getSortedUids(settlementUids) {
		addMember(uid)
	}

	for i := 0; i < len(settlements); i++ {
		balanceMap[settlements[i].FromUid].Sent += settlements[i].Amount
		balanceMap[settlements[i].ToUid].Received += settlements[i].Amount
	}

	for i := 0; i < len(balances); i++ {
		balances[i].Balance = balances[i].Paid - balances[i].FairShare + balances[i].Sent - balances[i].Received
	}

	return balances
}

// SuggestSettleUpPayments returns the payments which settle the given balances, it pays the largest debt to the largest credit again and again
// until every balance is 0, so there is at most one payment fewer than the members who are owed or owe anything.
// Equal amounts are taken in the order of the balances, so the result is always the same for the same balances.
func SuggestSettleUpPayments(balances []*models.BookSettleUpMemberBalance) []*models.BookSettleUpPayment {
	remaining := make([]int64, len(balances))

	for i := 0; i < len(balances); i++ {
		remaining[i] = balances[i].Balance
	}

	payments := make([]*models.BookSettleUpPayment, 0)

	for {
		creditorIndex := -1
		debtorIndex := -1

		for i := 0; i < len(remaining); i++ {
			if remaining[i] > 0 && (creditorIndex < 0 || remaining[i] > remaining[creditorIndex]) {
				creditorIndex = i
			} else if remaining[i] < 0 && (debtorIndex < 0 || remaining[i] < remaining[debtorIndex]) {
				debtorIndex = i
			}
		}

		if creditorIndex < 0 || debtorIndex < 0 {
			break
		}

		amount := remaining[creditorIndex]

		if -remaining[debtorIndex] < amount {
			amount = -remaining[debtorIndex]
		}

		payments = append(payments, &models.BookSettleUpPayment{
			FromUid: balances[debtorIndex].Uid,
			ToUid:   balances[creditorIndex].Uid,
			Amount:  amount,
		})

		remaining[creditorIndex] -= amount
		remaining[debtorIndex] += amount
	}

	return payments
}

func getSortedUids(uidMap map[int64]int64) []int64 {
	uids := make([]int64, 0, len(uidMap))

	for uid := range uidMap {
		uids = append(uids, uid)
	}

	sort.Slice(uids, func(i, j int) bool {
		return uids[i] < uids[j]
	})

	return uids
}
