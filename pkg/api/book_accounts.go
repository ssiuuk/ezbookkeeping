package api

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/log"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
)

// getAccountsOfBookWithUniqueNames returns copies of the book's accounts whose names tell apart the accounts shared to the book which have the same name as another account of the book,
// so that exported files keep each transaction's account and importing them back cannot pick a different account with the same name
func getAccountsOfBookWithUniqueNames(c *core.WebContext, accountService *services.AccountService, userService *services.UserService, accounts []*models.Account, bookId int64) []*models.Account {
	ownerNicknames := make(map[int64]string)

	for i := 0; i < len(accounts); i++ {
		ownerUid := accounts[i].Uid

		if ownerUid == bookId {
			continue
		}

		if _, exists := ownerNicknames[ownerUid]; exists {
			continue
		}

		owner, err := userService.GetUserById(c, ownerUid)

		if err != nil {
			log.Warnf(c, "[book_accounts.getAccountsOfBookWithUniqueNames] cannot get owner \"uid:%d\" of shared account in book \"id:%d\", because %s", ownerUid, bookId, err.Error())
			ownerNicknames[ownerUid] = ""
			continue
		}

		ownerNicknames[ownerUid] = owner.Nickname
	}

	return renameAccountsOfBookToUniqueNames(accountService, accounts, bookId, ownerNicknames)
}

func renameAccountsOfBookToUniqueNames(accountService *services.AccountService, accounts []*models.Account, bookId int64, ownerNicknames map[int64]string) []*models.Account {
	uniqueNames := accountService.GetUniqueAccountNamesOfBook(accounts, bookId, ownerNicknames)
	result := make([]*models.Account, len(accounts))

	for i := 0; i < len(accounts); i++ {
		account := *accounts[i]
		account.Name = uniqueNames[account.AccountId]
		result[i] = &account
	}

	return result
}
