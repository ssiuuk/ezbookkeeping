package api

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
)

// bookAccountNaming holds what is needed to tell apart the accounts of a book which have the same name
type bookAccountNaming struct {
	bookId         int64
	bookName       string
	ownerNicknames map[int64]string
}

// getBookAccountNaming returns the book name and the nicknames of the owners of the accounts shared to the book
func getBookAccountNaming(c *core.WebContext, userService *services.UserService, accounts []*models.Account, bookId int64) *bookAccountNaming {
	naming := &bookAccountNaming{
		bookId:         bookId,
		ownerNicknames: make(map[int64]string),
	}

	for i := 0; i < len(accounts); i++ {
		ownerUid := accounts[i].Uid

		if ownerUid == bookId {
			continue
		}

		if _, exists := naming.ownerNicknames[ownerUid]; exists {
			continue
		}

		owner, err := userService.GetUserById(c, ownerUid)

		if err != nil {
			log.Warnf(c, "[book_accounts.getBookAccountNaming] cannot get owner \"uid:%d\" of shared account in book \"id:%d\", because %s", ownerUid, bookId, err.Error())
			naming.ownerNicknames[ownerUid] = ""
			continue
		}

		naming.ownerNicknames[ownerUid] = owner.Nickname
	}

	// only a shared book can have accounts shared to it
	if len(naming.ownerNicknames) > 0 {
		book, err := services.Books.GetSharedBookById(c, bookId)

		if err != nil {
			log.Warnf(c, "[book_accounts.getBookAccountNaming] cannot get book \"id:%d\", because %s", bookId, err.Error())
		} else {
			naming.bookName = book.Name
		}
	}

	return naming
}

// renameAccountsToUniqueNames returns copies of the accounts named so that every account which has the same name as an account shared to the book gets an explicit name,
// so that exported files keep each transaction's account and importing them back cannot pick a different account with the same name
func renameAccountsToUniqueNames(accountService *services.AccountService, accounts []*models.Account, naming *bookAccountNaming) []*models.Account {
	uniqueNames := accountService.GetUniqueAccountNamesOfBook(accounts, naming.bookId, naming.bookName, naming.ownerNicknames)
	result := make([]*models.Account, len(accounts))

	for i := 0; i < len(accounts); i++ {
		account := *accounts[i]
		account.Name = uniqueNames[account.AccountId]
		result[i] = &account
	}

	return result
}

// getUniqueAccountNameMap returns the visible accounts keyed by the names renameAccountsToUniqueNames gives them, without the ambiguous original names
func getUniqueAccountNameMap(accountService *services.AccountService, accounts []*models.Account, naming *bookAccountNaming) map[string]*models.Account {
	return accountService.GetVisibleUniqueAccountNameMapOfBook(accounts, naming.bookId, naming.bookName, naming.ownerNicknames)
}
