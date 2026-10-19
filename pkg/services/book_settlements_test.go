package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mayswind/ezbookkeeping/pkg/models"
)

func getSettleUpBalanceMap(balances []*models.BookSettleUpMemberBalance) map[int64]*models.BookSettleUpMemberBalance {
	result := make(map[int64]*models.BookSettleUpMemberBalance, len(balances))

	for i := 0; i < len(balances); i++ {
		result[balances[i].Uid] = balances[i]
	}

	return result
}

func assertSettleUpBalancesAddUpToZero(t *testing.T, balances []*models.BookSettleUpMemberBalance) {
	total := int64(0)

	for i := 0; i < len(balances); i++ {
		total += balances[i].Balance
	}

	assert.Equal(t, int64(0), total)
}

func TestCalculateSettleUpBalances_TwoMembers(t *testing.T) {
	balances := CalculateSettleUpBalances([]int64{1, 2}, map[int64]int64{1: 3000, 2: 1000}, nil)

	assert.Equal(t, 2, len(balances))
	assert.Equal(t, int64(1), balances[0].Uid)
	assert.Equal(t, int64(3000), balances[0].Paid)
	assert.Equal(t, int64(2000), balances[0].FairShare)
	assert.Equal(t, int64(1000), balances[0].Balance)
	assert.Equal(t, int64(2), balances[1].Uid)
	assert.Equal(t, int64(1000), balances[1].Paid)
	assert.Equal(t, int64(2000), balances[1].FairShare)
	assert.Equal(t, int64(-1000), balances[1].Balance)

	payments := SuggestSettleUpPayments(balances)
	assert.Equal(t, 1, len(payments))
	assert.Equal(t, &models.BookSettleUpPayment{FromUid: 2, ToUid: 1, Amount: 1000}, payments[0])
}

func TestCalculateSettleUpBalances_MemberWhoPaidNothingStillShares(t *testing.T) {
	balances := CalculateSettleUpBalances([]int64{1, 2, 3}, map[int64]int64{1: 900}, nil)
	balanceMap := getSettleUpBalanceMap(balances)

	assert.Equal(t, int64(600), balanceMap[1].Balance)
	assert.Equal(t, int64(-300), balanceMap[2].Balance)
	assert.Equal(t, int64(-300), balanceMap[3].Balance)
	assert.Equal(t, int64(0), balanceMap[3].Paid)

	payments := SuggestSettleUpPayments(balances)
	assert.Equal(t, []*models.BookSettleUpPayment{
		{FromUid: 2, ToUid: 1, Amount: 300},
		{FromUid: 3, ToUid: 1, Amount: 300},
	}, payments)
}

func TestCalculateSettleUpBalances_RemainderGoesToFirstMembers(t *testing.T) {
	balances := CalculateSettleUpBalances([]int64{1, 2, 3}, map[int64]int64{2: 100}, nil)

	assert.Equal(t, int64(34), balances[0].FairShare)
	assert.Equal(t, int64(33), balances[1].FairShare)
	assert.Equal(t, int64(33), balances[2].FairShare)
	assert.Equal(t, int64(-34), balances[0].Balance)
	assert.Equal(t, int64(67), balances[1].Balance)
	assert.Equal(t, int64(-33), balances[2].Balance)
	assertSettleUpBalancesAddUpToZero(t, balances)
}

func TestCalculateSettleUpBalances_PayerOutsideSplitMembersAlsoShares(t *testing.T) {
	// member 9 has left the book but paid an expense, so the expenses are split among three
	balances := CalculateSettleUpBalances([]int64{1, 2}, map[int64]int64{1: 300, 9: 600}, nil)

	assert.Equal(t, 3, len(balances))
	assert.Equal(t, int64(9), balances[2].Uid)

	balanceMap := getSettleUpBalanceMap(balances)
	assert.Equal(t, int64(300), balanceMap[1].FairShare)
	assert.Equal(t, int64(0), balanceMap[1].Balance)
	assert.Equal(t, int64(-300), balanceMap[2].Balance)
	assert.Equal(t, int64(300), balanceMap[9].Balance)
}

func TestCalculateSettleUpBalances_SettlementsOffsetBalances(t *testing.T) {
	settlements := []*models.BookSettlement{
		{FromUid: 2, ToUid: 1, Amount: 600},
		{FromUid: 2, ToUid: 1, Amount: 400},
	}

	balances := CalculateSettleUpBalances([]int64{1, 2}, map[int64]int64{1: 3000, 2: 1000}, settlements)
	balanceMap := getSettleUpBalanceMap(balances)

	assert.Equal(t, int64(0), balanceMap[1].Balance)
	assert.Equal(t, int64(1000), balanceMap[1].Received)
	assert.Equal(t, int64(0), balanceMap[2].Balance)
	assert.Equal(t, int64(1000), balanceMap[2].Sent)
	assert.Equal(t, 0, len(SuggestSettleUpPayments(balances)))
}

func TestCalculateSettleUpBalances_OverpaidSettlementTurnsBalanceAround(t *testing.T) {
	settlements := []*models.BookSettlement{
		{FromUid: 2, ToUid: 1, Amount: 1500},
	}

	balances := CalculateSettleUpBalances([]int64{1, 2}, map[int64]int64{1: 2000}, settlements)
	balanceMap := getSettleUpBalanceMap(balances)

	assert.Equal(t, int64(-500), balanceMap[1].Balance)
	assert.Equal(t, int64(500), balanceMap[2].Balance)
	assert.Equal(t, []*models.BookSettleUpPayment{{FromUid: 1, ToUid: 2, Amount: 500}}, SuggestSettleUpPayments(balances))
}

func TestCalculateSettleUpBalances_SettlementOnlyMemberGetsNoShare(t *testing.T) {
	settlements := []*models.BookSettlement{
		{FromUid: 8, ToUid: 1, Amount: 100},
	}

	balances := CalculateSettleUpBalances([]int64{1, 2}, map[int64]int64{1: 200}, settlements)

	assert.Equal(t, 3, len(balances))
	assert.Equal(t, int64(8), balances[2].Uid)
	assert.Equal(t, int64(0), balances[2].FairShare)
	assert.Equal(t, int64(100), balances[2].Balance)
	assert.Equal(t, int64(0), balances[0].Balance)
	assert.Equal(t, int64(-100), balances[1].Balance)
	assertSettleUpBalancesAddUpToZero(t, balances)
}

func TestCalculateSettleUpBalances_NoExpense(t *testing.T) {
	balances := CalculateSettleUpBalances([]int64{1, 2}, nil, nil)

	assert.Equal(t, 2, len(balances))
	assert.Equal(t, int64(0), balances[0].Balance)
	assert.Equal(t, int64(0), balances[1].Balance)
	assert.Equal(t, 0, len(SuggestSettleUpPayments(balances)))

	assert.Equal(t, 0, len(CalculateSettleUpBalances(nil, nil, nil)))
}

func TestSuggestSettleUpPayments_LargestDebtorPaysLargestCreditorFirst(t *testing.T) {
	balances := []*models.BookSettleUpMemberBalance{
		{Uid: 1, Balance: 500},
		{Uid: 2, Balance: -100},
		{Uid: 3, Balance: 300},
		{Uid: 4, Balance: -700},
	}

	payments := SuggestSettleUpPayments(balances)

	assert.Equal(t, []*models.BookSettleUpPayment{
		{FromUid: 4, ToUid: 1, Amount: 500},
		{FromUid: 4, ToUid: 3, Amount: 200},
		{FromUid: 2, ToUid: 3, Amount: 100},
	}, payments)

	// the balances themselves are left as they are
	assert.Equal(t, int64(500), balances[0].Balance)
	assert.Equal(t, int64(-700), balances[3].Balance)
}

func TestSuggestSettleUpPayments_EqualAmountsTakenInOrder(t *testing.T) {
	balances := []*models.BookSettleUpMemberBalance{
		{Uid: 5, Balance: -200},
		{Uid: 6, Balance: 200},
		{Uid: 7, Balance: -200},
		{Uid: 8, Balance: 200},
	}

	payments := SuggestSettleUpPayments(balances)

	assert.Equal(t, []*models.BookSettleUpPayment{
		{FromUid: 5, ToUid: 6, Amount: 200},
		{FromUid: 7, ToUid: 8, Amount: 200},
	}, payments)
}

func TestSuggestSettleUpPayments_SettlesEveryBalance(t *testing.T) {
	balances := CalculateSettleUpBalances([]int64{1, 2, 3, 4, 5}, map[int64]int64{1: 12345, 2: 678, 4: 9999}, []*models.BookSettlement{
		{FromUid: 3, ToUid: 1, Amount: 1000},
	})
	assertSettleUpBalancesAddUpToZero(t, balances)

	payments := SuggestSettleUpPayments(balances)
	assert.LessOrEqual(t, len(payments), len(balances)-1)

	remaining := make(map[int64]int64)

	for i := 0; i < len(balances); i++ {
		remaining[balances[i].Uid] = balances[i].Balance
	}

	for i := 0; i < len(payments); i++ {
		assert.Greater(t, payments[i].Amount, int64(0))
		remaining[payments[i].FromUid] += payments[i].Amount
		remaining[payments[i].ToUid] -= payments[i].Amount
	}

	for uid, amount := range remaining {
		assert.Equal(t, int64(0), amount, "uid %d", uid)
	}
}
