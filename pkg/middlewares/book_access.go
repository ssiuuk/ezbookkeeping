package middlewares

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
)

const bookIdHeaderName = "X-Book-Id"
const bookIdQueryStringParam = "book_id"

// BookAccess sets the book which current request operates on, when the user works in a book shared with them
func BookAccess(config *settings.Config) core.MiddlewareHandlerFunc {
	return func(c *core.WebContext) {
		claims := c.GetTokenClaims()

		if claims == nil {
			c.Next()
			return
		}

		textualBookId := c.GetHeader(bookIdHeaderName)

		if textualBookId == "" {
			textualBookId = c.Query(bookIdQueryStringParam)
		}

		if textualBookId == "" {
			c.Next()
			return
		}

		bookUid, err := utils.StringToInt64(textualBookId)

		if err != nil || bookUid <= 0 {
			log.Warnf(c, "[book_access.BookAccess] book id \"%s\" is invalid", textualBookId)
			utils.PrintJsonErrorResult(c, errs.ErrBookIdInvalid)
			return
		}

		if bookUid == claims.Uid {
			c.Next()
			return
		}

		canAccess, err := services.Books.CanAccessBook(c, claims.Uid, bookUid)

		if err != nil {
			log.Errorf(c, "[book_access.BookAccess] failed to check whether user \"uid:%d\" can access book \"uid:%d\", because %s", claims.Uid, bookUid, err.Error())
			utils.PrintJsonErrorResult(c, errs.Or(err, errs.ErrOperationFailed))
			return
		}

		if !canAccess {
			log.Warnf(c, "[book_access.BookAccess] user \"uid:%d\" is not a member of book \"uid:%d\"", claims.Uid, bookUid)
			utils.PrintJsonErrorResult(c, errs.ErrBookAccessDenied)
			return
		}

		c.SetCurrentBookUid(bookUid)
		c.Next()
	}
}
