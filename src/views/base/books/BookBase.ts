import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useBooksStore } from '@/stores/book.ts';

import type { BookInfoResponse } from '@/models/book.ts';

export function useBookBase() {
    const { tt } = useI18n();

    const booksStore = useBooksStore();

    const allBooks = computed<BookInfoResponse[]>(() => booksStore.allBooks);
    const currentBook = computed<BookInfoResponse | undefined>(() => booksStore.currentBook);
    const isInSharedBook = computed<boolean>(() => booksStore.isInSharedBook);
    const showBookSwitcher = computed<boolean>(() => booksStore.hasSharedBooks || booksStore.isInSharedBook);
    const currentBookDisplayName = computed<string>(() => currentBook.value ? getBookDisplayName(currentBook.value) : tt('My Book'));

    function getBookDisplayName(book: BookInfoResponse): string {
        if (book.isOwner) {
            return tt('My Book');
        }

        return tt('format.misc.bookOfUser', { name: book.ownerNickname || book.ownerUsername });
    }

    function isCurrentBook(book: BookInfoResponse): boolean {
        return currentBook.value?.id === book.id;
    }

    function switchBook(book: BookInfoResponse): void {
        if (isCurrentBook(book)) {
            return;
        }

        booksStore.switchBook(book);
    }

    return {
        // computed states
        allBooks,
        currentBook,
        isInSharedBook,
        showBookSwitcher,
        currentBookDisplayName,
        // functions
        getBookDisplayName,
        isCurrentBook,
        switchBook
    };
}
