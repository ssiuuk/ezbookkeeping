import { ref, computed } from 'vue';

import { useBookBase } from './BookBase.ts';

import { useBooksStore } from '@/stores/book.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useTransactionTagsStore } from '@/stores/transactionTag.ts';
import { useProjectsStore } from '@/stores/project.ts';

import type { BookInfoResponse } from '@/models/book.ts';

import { setActiveBookIdOverride, hasActiveBookIdOverride } from '@/lib/services.ts';

// lets a new transaction be added to another book without switching the whole app to it
export function useEntryBookSelectionBase() {
    const { allBooks, getBookDisplayName } = useBookBase();

    const booksStore = useBooksStore();
    const accountsStore = useAccountsStore();
    const transactionCategoriesStore = useTransactionCategoriesStore();
    const transactionTagsStore = useTransactionTagsStore();
    const projectsStore = useProjectsStore();

    const entryBookId = ref<string | undefined>(undefined);

    const showEntryBookSelection = computed<boolean>(() => allBooks.value.length > 1);
    const entryBook = computed<BookInfoResponse | undefined>(() => allBooks.value.find(book => book.id === entryBookId.value) ?? booksStore.currentBook);
    const entryBookDisplayName = computed<string>(() => entryBook.value ? getBookDisplayName(entryBook.value) : '');

    function ignoreUpToDate(promise: Promise<unknown>): Promise<unknown> {
        // a forced reload rejects when the data has not changed, which is fine here
        return promise.catch(error => error && error.isUpToDate ? undefined : Promise.reject(error));
    }

    function reloadBookData(): Promise<unknown> {
        return Promise.all([
            ignoreUpToDate(accountsStore.loadAllAccounts({ force: true })),
            ignoreUpToDate(transactionCategoriesStore.loadAllCategories({ force: true })),
            ignoreUpToDate(transactionTagsStore.loadAllTags({ force: true })),
            projectsStore.loadAllProjects({ force: true })
        ]);
    }

    function initEntryBook(): void {
        entryBookId.value = booksStore.currentBook?.id;
    }

    function changeEntryBook(book: BookInfoResponse): Promise<unknown> {
        if (book.id === entryBook.value?.id) {
            return Promise.resolve();
        }

        entryBookId.value = book.id;

        if (book.id === booksStore.currentBook?.id) {
            setActiveBookIdOverride(undefined);
        } else {
            setActiveBookIdOverride(book.isOwner ? null : book.id);
        }

        return reloadBookData();
    }

    function restoreEntryBook(): Promise<unknown> {
        initEntryBook();

        if (!hasActiveBookIdOverride()) {
            return Promise.resolve();
        }

        setActiveBookIdOverride(undefined);
        return reloadBookData();
    }

    return {
        // states
        entryBookId,
        // computed states
        allBooks,
        showEntryBookSelection,
        entryBook,
        entryBookDisplayName,
        // functions
        getBookDisplayName,
        initEntryBook,
        changeEntryBook,
        restoreEntryBook
    };
}
