import { ref, computed } from 'vue';
import { defineStore } from 'pinia';

import type {
    BookInfoResponse,
    BookMemberInfoResponse
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

        return allBooks.value.find(book => book.isOwner);
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

    function switchBook(book: BookInfoResponse): void {
        setCurrentBookId(book.isOwner ? null : book.id);
        clearUserTransactionDraft();
        window.location.reload();
    }

    function loadAllBooks(): Promise<BookInfoResponse[]> {
        return handleApiResponse(services.getBooks(), 'failed to load book list', 'Unable to retrieve book list').then(books => {
            allBooks.value = books;

            if (currentBookId.value && !books.some(book => book.id === currentBookId.value && !book.isOwner)) {
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

    function addBookMember(username: string): Promise<BookMemberInfoResponse> {
        return handleApiResponse(services.addBookMember({ username }), 'failed to add book member', 'Unable to add book member').then(member => {
            currentBookMembers.value.push(member);
            return member;
        });
    }

    function removeBookMember(memberId: string): Promise<boolean> {
        return handleApiResponse(services.removeBookMember({ memberId }), 'failed to remove book member', 'Unable to remove book member').then(result => {
            currentBookMembers.value = currentBookMembers.value.filter(member => member.id !== memberId);
            return result;
        });
    }

    function leaveBook(bookId: string): Promise<boolean> {
        return handleApiResponse(services.leaveBook({ bookId }), 'failed to leave book', 'Unable to leave book').then(result => {
            allBooks.value = allBooks.value.filter(book => book.id !== bookId);
            return result;
        });
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
        initBooks,
        switchBook,
        loadAllBooks,
        loadCurrentBookMembers,
        resetBooks,
        addBookMember,
        removeBookMember,
        leaveBook
    };
});
