package models

// BookSettlement represents a payment between two members of a shared book which settles what one owes the other, stored in database
type BookSettlement struct {
	SettlementId    int64  `xorm:"PK"`
	BookId          int64  `xorm:"INDEX(IDX_book_settlement_book_id_deleted_time) NOT NULL"`
	Deleted         bool   `xorm:"INDEX(IDX_book_settlement_book_id_deleted_time) NOT NULL"`
	FromUid         int64  `xorm:"NOT NULL"`
	ToUid           int64  `xorm:"NOT NULL"`
	Amount          int64  `xorm:"NOT NULL"`
	Currency        string `xorm:"VARCHAR(3) NOT NULL"`
	SettlementTime  int64  `xorm:"INDEX(IDX_book_settlement_book_id_deleted_time) NOT NULL"`
	CreatedByUid    int64  `xorm:"NOT NULL"`
	Comment         string `xorm:"VARCHAR(255) NOT NULL"`
	CreatedUnixTime int64
	UpdatedUnixTime int64
	DeletedUnixTime int64
}

// BookSettleUpRequest represents all parameters of the settle up and settlement list requests of current shared book,
// the times are unix times in seconds, a time of 0 means the range is not limited on that side
type BookSettleUpRequest struct {
	StartTime int64 `form:"start_time" binding:"min=0"`
	EndTime   int64 `form:"end_time" binding:"min=0"`
}

// BookSettlementCreateRequest represents all parameters of recording a settlement in current shared book,
// a time of 0 means now
type BookSettlementCreateRequest struct {
	FromUid  int64  `json:"fromId,string" binding:"required,min=1"`
	ToUid    int64  `json:"toId,string" binding:"required,min=1"`
	Amount   int64  `json:"amount" binding:"required,min=1,max=99999999999"`
	Currency string `json:"currency" binding:"required,len=3,validCurrency"`
	Time     int64  `json:"time" binding:"min=0"`
	Comment  string `json:"comment" binding:"max=255"`
}

// BookSettlementDeleteRequest represents all parameters of settlement deleting request
type BookSettlementDeleteRequest struct {
	Id int64 `json:"id,string" binding:"required,min=1"`
}

// BookSettlementInfoResponse represents a view-object of settlement
type BookSettlementInfoResponse struct {
	Id           int64  `json:"id,string"`
	FromUid      int64  `json:"fromId,string"`
	ToUid        int64  `json:"toId,string"`
	Amount       int64  `json:"amount"`
	Currency     string `json:"currency"`
	Time         int64  `json:"time"`
	CreatedByUid int64  `json:"createdById,string"`
	Comment      string `json:"comment"`
	CreatedAt    int64  `json:"createdAt"`
}

// ToBookSettlementInfoResponse returns a view-object according to database model
func (s *BookSettlement) ToBookSettlementInfoResponse() *BookSettlementInfoResponse {
	return &BookSettlementInfoResponse{
		Id:           s.SettlementId,
		FromUid:      s.FromUid,
		ToUid:        s.ToUid,
		Amount:       s.Amount,
		Currency:     s.Currency,
		Time:         s.SettlementTime,
		CreatedByUid: s.CreatedByUid,
		Comment:      s.Comment,
		CreatedAt:    s.CreatedUnixTime,
	}
}

// BookSettleUpMemberResponse represents a view-object of a member who takes part in the settle up of a shared book
type BookSettleUpMemberResponse struct {
	Id       int64  `json:"id,string"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	IsOwner  bool   `json:"isOwner"`
	IsFormer bool   `json:"isFormer"`
}

// BookSettleUpMemberBalance represents what a member paid, their fair share and what they are owed (a positive balance) or owe (a negative balance) in one currency
type BookSettleUpMemberBalance struct {
	Uid       int64 `json:"memberId,string"`
	Paid      int64 `json:"paid"`
	FairShare int64 `json:"fairShare"`
	Sent      int64 `json:"settlementsSent"`
	Received  int64 `json:"settlementsReceived"`
	Balance   int64 `json:"balance"`
}

// BookSettleUpPayment represents a payment which a member should make to another member to settle up
type BookSettleUpPayment struct {
	FromUid int64 `json:"fromId,string"`
	ToUid   int64 `json:"toId,string"`
	Amount  int64 `json:"amount"`
}

// BookSettleUpCurrencyResponse represents the settle up of a shared book in one currency
type BookSettleUpCurrencyResponse struct {
	Currency          string                       `json:"currency"`
	TotalExpense      int64                        `json:"totalExpense"`
	Balances          []*BookSettleUpMemberBalance `json:"balances"`
	SuggestedPayments []*BookSettleUpPayment       `json:"suggestedPayments"`
}

// BookSettleUpResponse represents a view-object of the settle up of current shared book,
// amounts are not converted between currencies, each currency is settled up separately
type BookSettleUpResponse struct {
	DefaultCurrency string                          `json:"defaultCurrency"`
	StartTime       int64                           `json:"startTime"`
	EndTime         int64                           `json:"endTime"`
	Members         []*BookSettleUpMemberResponse   `json:"members"`
	Currencies      []*BookSettleUpCurrencyResponse `json:"currencies"`
}

// BookSettleUpAccountExpense represents the total expense of a shared book which was paid from one account
type BookSettleUpAccountExpense struct {
	AccountId   int64 `xorm:"account_id"`
	TotalAmount int64 `xorm:"total_amount"`
}
