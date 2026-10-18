package models

// SharedBook represents a book which is created on top of a user's own book and can be shared with other users
type SharedBook struct {
	BookId          int64  `xorm:"PK"`
	OwnerUid        int64  `xorm:"INDEX(IDX_shared_book_owner_uid_deleted) NOT NULL"`
	Deleted         bool   `xorm:"INDEX(IDX_shared_book_owner_uid_deleted) NOT NULL"`
	Name            string `xorm:"VARCHAR(64) NOT NULL"`
	CreatedUnixTime int64
	UpdatedUnixTime int64
	DeletedUnixTime int64
}

// BookMember represents a user who can access and edit a shared book
type BookMember struct {
	BookUid         int64 `xorm:"PK"`
	MemberUid       int64 `xorm:"PK INDEX(IDX_book_member_member_uid)"`
	CreatedUnixTime int64
}

// SharedBookAccount represents an account of a book member which can be used in a shared book
type SharedBookAccount struct {
	BookId          int64 `xorm:"PK"`
	AccountId       int64 `xorm:"PK INDEX(IDX_shared_book_account_account_id)"`
	OwnerUid        int64 `xorm:"NOT NULL"`
	CreatedUnixTime int64
}

// BookCreateRequest represents all parameters of shared book creation request
type BookCreateRequest struct {
	Name string `json:"name" binding:"required,notBlank,max=64"`
}

// BookModifyRequest represents all parameters of shared book modification request
type BookModifyRequest struct {
	Id   int64  `json:"id,string" binding:"required,min=1"`
	Name string `json:"name" binding:"required,notBlank,max=64"`
}

// BookDeleteRequest represents all parameters of shared book deletion request
type BookDeleteRequest struct {
	Id int64 `json:"id,string" binding:"required,min=1"`
}

// BookMemberAddRequest represents all parameters of book member adding request
type BookMemberAddRequest struct {
	Username string `json:"username" binding:"required,notBlank,max=32,validUsername"`
}

// BookMemberRemoveRequest represents all parameters of book member removing request
type BookMemberRemoveRequest struct {
	MemberId int64 `json:"memberId,string" binding:"required,min=1"`
}

// BookLeaveRequest represents all parameters of leaving a shared book request
type BookLeaveRequest struct {
	BookId int64 `json:"bookId,string" binding:"required,min=1"`
}

// BookAccountShareRequest represents all parameters of sharing an account to or unsharing an account from current shared book
type BookAccountShareRequest struct {
	AccountId int64 `json:"accountId,string" binding:"required,min=1"`
}

// BookInfoResponse represents a view-object of a book which current user can access
type BookInfoResponse struct {
	Id            int64  `json:"id,string"`
	Name          string `json:"name"`
	OwnerUsername string `json:"ownerUsername"`
	OwnerNickname string `json:"ownerNickname"`
	IsPersonal    bool   `json:"isPersonal"`
	IsOwner       bool   `json:"isOwner"`
}

// BookMemberInfoResponse represents a view-object of book member
type BookMemberInfoResponse struct {
	Id        int64  `json:"id,string"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	IsOwner   bool   `json:"isOwner"`
	CreatedAt int64  `json:"createdAt"`
}

// BookSharedAccountInfoResponse represents a view-object of an account shared to current book
type BookSharedAccountInfoResponse struct {
	AccountId     int64  `json:"accountId,string"`
	OwnerUid      int64  `json:"ownerId,string"`
	OwnerUsername string `json:"ownerUsername"`
	OwnerNickname string `json:"ownerNickname"`
}
