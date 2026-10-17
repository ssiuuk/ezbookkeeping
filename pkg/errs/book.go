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
)
