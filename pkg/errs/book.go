package errs

import "net/http"

// Error codes related to shared books
var (
	ErrBookIdInvalid                     = NewNormalError(NormalSubcategoryBook, 0, http.StatusBadRequest, "book id is invalid")
	ErrBookAccessDenied                  = NewNormalError(NormalSubcategoryBook, 1, http.StatusForbidden, "you are not a member of this book")
	ErrBookMemberNotFound                = NewNormalError(NormalSubcategoryBook, 2, http.StatusBadRequest, "book member not found")
	ErrBookMemberAlreadyExists           = NewNormalError(NormalSubcategoryBook, 3, http.StatusBadRequest, "this user is already a member of your book")
	ErrCannotAddYourselfAsBookMember     = NewNormalError(NormalSubcategoryBook, 4, http.StatusBadRequest, "you cannot add yourself as a member of your own book")
	ErrBookMemberCannotManageMembers     = NewNormalError(NormalSubcategoryBook, 5, http.StatusForbidden, "only the book owner can manage book members")
	ErrOperationNotAvailableInSharedBook = NewNormalError(NormalSubcategoryBook, 6, http.StatusBadRequest, "this operation is only available in your own book")
	ErrSharedBookNotFound                = NewNormalError(NormalSubcategoryBook, 7, http.StatusBadRequest, "shared book not found")
	ErrOnlyBookOwnerCanModifyBook        = NewNormalError(NormalSubcategoryBook, 8, http.StatusForbidden, "only the book owner can modify or delete this book")
	ErrSharedBookNotEmpty                = NewNormalError(NormalSubcategoryBook, 9, http.StatusBadRequest, "remove all transactions in this book before deleting it")
	ErrNotInSharedBook                   = NewNormalError(NormalSubcategoryBook, 10, http.StatusBadRequest, "this operation is only available in a shared book")
	ErrAccountCannotBeShared             = NewNormalError(NormalSubcategoryBook, 11, http.StatusBadRequest, "only your own accounts without sub-accounts can be shared")
	ErrAccountAlreadySharedToBook        = NewNormalError(NormalSubcategoryBook, 12, http.StatusBadRequest, "this account is already shared to this book")
	ErrSharedAccountNotFound             = NewNormalError(NormalSubcategoryBook, 13, http.StatusBadRequest, "shared account not found")
	ErrSharedAccountInUse                = NewNormalError(NormalSubcategoryBook, 14, http.StatusBadRequest, "this account is used by transactions in the shared book")
	ErrAccountSharedToBook               = NewNormalError(NormalSubcategoryBook, 15, http.StatusBadRequest, "this account is shared to a book, stop sharing it first")
	ErrCannotManageSharedAccount         = NewNormalError(NormalSubcategoryBook, 16, http.StatusForbidden, "you can only stop sharing your own accounts")
	ErrSharedAccountRevoked              = NewNormalError(NormalSubcategoryBook, 17, http.StatusBadRequest, "this account is no longer shared to this book, its transactions cannot be added, changed or deleted")
)
