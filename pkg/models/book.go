package models

// BookMember represents a user who can access and edit another user's book
type BookMember struct {
	BookUid         int64 `xorm:"PK"`
	MemberUid       int64 `xorm:"PK INDEX(IDX_book_member_member_uid)"`
	CreatedUnixTime int64
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

// BookInfoResponse represents a view-object of a book which current user can access
type BookInfoResponse struct {
	Id            int64  `json:"id,string"`
	OwnerUsername string `json:"ownerUsername"`
	OwnerNickname string `json:"ownerNickname"`
	IsOwner       bool   `json:"isOwner"`
}

// BookMemberInfoResponse represents a view-object of book member
type BookMemberInfoResponse struct {
	Id        int64  `json:"id,string"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	CreatedAt int64  `json:"createdAt"`
}
