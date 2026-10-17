import { ref, computed } from 'vue';

import { useBookBase } from '@/views/base/books/BookBase.ts';

import { useBooksStore } from '@/stores/book.ts';

import type { BookInfoResponse, BookMemberInfoResponse } from '@/models/book.ts';
import type { AccountInfoResponse } from '@/models/account.ts';

import { AccountType } from '@/core/account.ts';

import services from '@/lib/services.ts';
import { handleApiResponse } from '@/lib/api.ts';

export interface SharedBookAccountItem {
    readonly accountId: string;
    readonly name: string;
    readonly ownerName: string;
    readonly isMine: boolean;
}

export interface SharedBookDetail {
    loading: boolean;
    members: BookMemberInfoResponse[];
    sharedAccounts: SharedBookAccountItem[];
}

export function useSharedBooksPageBase() {
    const {
        allBooks,
        isCurrentBook,
        switchBook
    } = useBookBase();

    const booksStore = useBooksStore();

    const loading = ref<boolean>(true);
    const myAccounts = ref<AccountInfoResponse[]>([]);
    const bookDetails = ref<Record<string, SharedBookDetail>>({});

    const personalBook = computed<BookInfoResponse | undefined>(() => allBooks.value.find(book => book.isPersonal));
    const sharedBooks = computed<BookInfoResponse[]>(() => allBooks.value.filter(book => !book.isPersonal));

    function getBookDetail(book: BookInfoResponse): SharedBookDetail {
        return bookDetails.value[book.id] || { loading: true, members: [], sharedAccounts: [] };
    }

    function isAccountShared(book: BookInfoResponse, accountId: string): boolean {
        return getBookDetail(book).sharedAccounts.some(account => account.accountId === accountId);
    }

    function loadMyAccounts(): Promise<AccountInfoResponse[]> {
        return handleApiResponse(services.getAllAccounts({ visibleOnly: true }, { bookId: null }), 'failed to load accounts', 'Unable to retrieve account list').then(accounts => {
            // only accounts without sub-accounts can be shared to a book
            myAccounts.value = accounts.filter(account => account.type === AccountType.SingleAccount.type);
            return accounts;
        });
    }

    function loadBookDetail(book: BookInfoResponse): Promise<void> {
        const detail: SharedBookDetail = bookDetails.value[book.id] || { loading: true, members: [], sharedAccounts: [] };
        detail.loading = true;
        bookDetails.value[book.id] = detail;

        return Promise.all([
            booksStore.loadBookMembers(book.id),
            booksStore.loadBookSharedAccounts(book.id),
            handleApiResponse(services.getAllAccounts({ visibleOnly: false }, { bookId: book.id }), 'failed to load accounts', 'Unable to retrieve account list')
        ]).then(([members, sharedAccounts, bookAccounts]) => {
            const accountNames: Record<string, string> = {};
            const myAccountIds: Record<string, boolean> = {};

            for (const account of bookAccounts) {
                accountNames[account.id] = account.name;
            }

            for (const account of myAccounts.value) {
                myAccountIds[account.id] = true;
            }

            bookDetails.value[book.id] = {
                loading: false,
                members: members,
                sharedAccounts: sharedAccounts.map(sharedAccount => ({
                    accountId: sharedAccount.accountId,
                    name: accountNames[sharedAccount.accountId] || '',
                    ownerName: sharedAccount.ownerNickname || sharedAccount.ownerUsername,
                    isMine: !!myAccountIds[sharedAccount.accountId]
                }))
            };
        }).catch(error => {
            detail.loading = false;
            throw error;
        });
    }

    function reloadAll(): Promise<void> {
        loading.value = true;

        return Promise.all([
            booksStore.loadAllBooks(),
            loadMyAccounts()
        ]).then(() => {
            return Promise.all(sharedBooks.value.map(book => loadBookDetail(book)));
        }).then(() => {
            loading.value = false;
        }).catch(error => {
            loading.value = false;
            throw error;
        });
    }

    function createBook(name: string): Promise<BookInfoResponse> {
        return booksStore.createBook(name).then(book => {
            return loadBookDetail(book).then(() => book);
        });
    }

    function addMember(book: BookInfoResponse, username: string): Promise<void> {
        return booksStore.addBookMember(book.id, username).then(() => loadBookDetail(book));
    }

    function removeMember(book: BookInfoResponse, member: BookMemberInfoResponse): Promise<void> {
        return booksStore.removeBookMember(book.id, member.id).then(() => loadBookDetail(book));
    }

    function setAccountShared(book: BookInfoResponse, accountId: string, shared: boolean): Promise<void> {
        const promise = shared ? booksStore.shareAccountToBook(book.id, accountId) : booksStore.unshareAccountFromBook(book.id, accountId);
        return promise.then(() => loadBookDetail(book));
    }

    return {
        // states
        loading,
        myAccounts,
        // computed states
        personalBook,
        sharedBooks,
        // functions
        isCurrentBook,
        switchBook,
        getBookDetail,
        isAccountShared,
        reloadAll,
        createBook,
        renameBook: booksStore.renameBook,
        deleteBook: booksStore.deleteBook,
        leaveBook: booksStore.leaveBook,
        addMember,
        removeMember,
        setAccountShared
    };
}
