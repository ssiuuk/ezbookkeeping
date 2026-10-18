import { ref, computed } from 'vue';
import { defineStore } from 'pinia';

import type {
    BookInfoResponse,
    BookMemberInfoResponse,
    BookSharedAccountInfoResponse
} from '@/models/book.ts';

import { getCurrentBookId, setCurrentBookId, clearUserTransactionDraft } from '@/lib/userstate.ts';
import logger from '@/lib/logger.ts';
import services from '@/lib/services.ts';
import { handleApiResponse } from '@/lib/api.ts';

export const useBooksStore = defineStore('books', () => {
    const allBooks = ref<BookInfoResponse[]>([]);
    const currentBookMembers = ref<BookMemberInfoResponse[]>([]);
    const currentBookId = ref<string | null>(getCurrentBookId());

    const currentBook = computed<BookInfoResponse | undefined>(() => {
        if (currentBookId.value) {
            return allBooks.value.find(book => book.id === currentBookId.value);
        }

        return allBooks.value.find(book => book.isPersonal);
    });

    const isInSharedBook = computed<boolean>(() => !!currentBookId.value);
    const hasSharedBooks = computed<boolean>(() => allBooks.value.length > 1);

    function getMemberDisplayName(uid: string | undefined): string {
        if (!uid) {
            return '';
        }

        const member = currentBookMembers.value.find(item => item.id === uid);
        return member ? (member.nickname || member.username) : '';
    }

    function getRequestBookId(book: BookInfoResponse): string | null {
        return book.isPersonal ? null : book.id;
    }

    function switchBook(book: BookInfoResponse): void {
        setCurrentBookId(getRequestBookId(book));
        clearUserTransactionDraft();
        window.location.reload();
    }

    function loadAllBooks(): Promise<BookInfoResponse[]> {
        return handleApiResponse(services.getBooks(), 'failed to load book list', 'Unable to retrieve book list').then(books => {
            allBooks.value = books;

            if (currentBookId.value && !books.some(book => book.id === currentBookId.value && !book.isPersonal)) {
                // the shared book is no longer accessible, go back to the user's own book
                setCurrentBookId(null);
                currentBookId.value = null;
            }

            return books;
        });
    }

    function loadCurrentBookMembers(): Promise<BookMemberInfoResponse[]> {
        return handleApiResponse(services.getBookMembers(), 'failed to load book members', 'Unable to retrieve book members').then(members => {
            currentBookMembers.value = members;
            return members;
        });
    }

    function resetBooks(): void {
        allBooks.value = [];
        currentBookMembers.value = [];
        currentBookId.value = getCurrentBookId();
    }

    function initBooks(): void {
        Promise.all([
            loadAllBooks(),
            loadCurrentBookMembers()
        ]).catch(error => {
            logger.warn('failed to initialize books', error);
        });
    }

    function createBook(name: string): Promise<BookInfoResponse> {
        return handleApiResponse(services.addBook({ name }), 'failed to create shared book', 'Unable to create shared book').then(book => {
            allBooks.value.push(book);
            return book;
        });
    }

    function renameBook(bookId: string, name: string): Promise<boolean> {
        return handleApiResponse(services.modifyBook({ id: bookId, name }), 'failed to rename shared book', 'Unable to rename shared book').then(result => {
            allBooks.value = allBooks.value.map(book => book.id === bookId ? { ...book, name } : book);
            return result;
        });
    }

    function deleteBook(bookId: string): Promise<boolean> {
        return handleApiResponse(services.deleteBook({ id: bookId }), 'failed to delete shared book', 'Unable to delete shared book').then(result => {
            allBooks.value = allBooks.value.filter(book => book.id !== bookId);
            return result;
        });
    }

    function loadBookMembers(bookId: string): Promise<BookMemberInfoResponse[]> {
        return handleApiResponse(services.getBookMembers({ bookId }), 'failed to load book members', 'Unable to retrieve book members');
    }

    function addBookMember(bookId: string, username: string): Promise<BookMemberInfoResponse> {
        return handleApiResponse(services.addBookMember({ username }, { bookId }), 'failed to add book member', 'Unable to add book member');
    }

    function removeBookMember(bookId: string, memberId: string): Promise<boolean> {
        return handleApiResponse(services.removeBookMember({ memberId }, { bookId }), 'failed to remove book member', 'Unable to remove book member');
    }

    function leaveBook(bookId: string): Promise<boolean> {
        return handleApiResponse(services.leaveBook({ bookId }), 'failed to leave book', 'Unable to leave book').then(result => {
            allBooks.value = allBooks.value.filter(book => book.id !== bookId);
            return result;
        });
    }

    function loadBookSharedAccounts(bookId: string): Promise<BookSharedAccountInfoResponse[]> {
        return handleApiResponse(services.getBookSharedAccounts({ bookId }), 'failed to load shared accounts', 'Unable to retrieve shared accounts');
    }

    function shareAccountToBook(bookId: string, accountId: string): Promise<boolean> {
        return handleApiResponse(services.shareAccountToBook({ accountId }, { bookId }), 'failed to share account', 'Unable to share account');
    }

    function unshareAccountFromBook(bookId: string, accountId: string): Promise<boolean> {
        return handleApiResponse(services.unshareAccountFromBook({ accountId }, { bookId }), 'failed to stop sharing account', 'Unable to stop sharing account');
    }

    return {
        // states
        allBooks,
        currentBookMembers,
        currentBookId,
        // computed states
        currentBook,
        isInSharedBook,
        hasSharedBooks,
        // functions
        getMemberDisplayName,
        getRequestBookId,
        initBooks,
        switchBook,
        loadAllBooks,
        loadCurrentBookMembers,
        resetBooks,
        createBook,
        renameBook,
        deleteBook,
        loadBookMembers,
        addBookMember,
        removeBookMember,
        leaveBook,
        loadBookSharedAccounts,
        shareAccountToBook,
        unshareAccountFromBook
    };
});
