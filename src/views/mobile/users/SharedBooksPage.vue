<template>
    <f7-page @page:afterin="onPageAfterIn">
        <f7-navbar :title="tt('Shared Books')" :back-link="tt('Back')"></f7-navbar>

        <f7-block class="margin-top-half margin-bottom-half">
            <small>{{ tt('A shared book sits next to your own book. Add family members to it, and share the accounts they may spend from. Your own book stays private.') }}</small>
        </f7-block>

        <f7-block-title>{{ tt('Switch Book') }}</f7-block-title>
        <f7-list strong inset dividers :class="{ 'skeleton-text': loading }">
            <f7-list-item radio
                          :title="tt('My Book')"
                          :checked="isCurrentBook(personalBook)"
                          v-if="personalBook"
                          @change="switchBook(personalBook)">
            </f7-list-item>
            <f7-list-item radio
                          :key="book.id"
                          :title="book.name"
                          :footer="book.ownerNickname || book.ownerUsername"
                          :checked="isCurrentBook(book)"
                          v-for="book in sharedBooks"
                          @change="switchBook(book)">
            </f7-list-item>
        </f7-list>

        <f7-list strong inset dividers>
            <f7-list-input
                type="text"
                autocomplete="off"
                clear-button
                :label="tt('Book Name')"
                :placeholder="tt('Name of the new shared book, e.g. Family')"
                v-model:value="newBookName"
            ></f7-list-input>
            <f7-list-button :class="{ 'disabled': !newBookName.trim() }" @click="create">{{ tt('Create Shared Book') }}</f7-list-button>
        </f7-list>

        <template :key="book.id" v-for="book in sharedBooks">
            <f7-block-title>{{ book.name }}</f7-block-title>
            <f7-list strong inset dividers :class="{ 'skeleton-text': getBookDetail(book).loading }">
                <f7-list-item :title="member.nickname || member.username"
                              :footer="member.username"
                              :key="member.id"
                              v-for="member in getBookDetail(book).members">
                    <template #after>
                        <span v-if="member.isOwner || !book.isOwner">{{ member.isOwner ? tt('Owner') : tt('Member') }}</span>
                        <f7-link color="red" v-else @click="removeBookMember(book, member)">{{ tt('Remove') }}</f7-link>
                    </template>
                </f7-list-item>
                <f7-list-button v-if="book.isOwner" @click="addBookMember(book)">{{ tt('Add Member') }}</f7-list-button>
            </f7-list>

            <f7-block-header>{{ tt('Members of this book can record transactions with these accounts, and see their balances.') }}</f7-block-header>
            <f7-list strong inset dividers>
                <f7-list-item :title="account.name"
                              :key="account.accountId"
                              v-for="account in getBookDetail(book).sharedAccounts.filter(item => !item.isMine)">
                    <template #after>
                        <f7-link color="red" v-if="book.isOwner" @click="toggleAccount(book, account.accountId, false)">{{ tt('Stop Sharing') }}</f7-link>
                    </template>
                </f7-list-item>
                <f7-list-item checkbox
                              :key="account.id"
                              :title="account.name"
                              :checked="isAccountShared(book, account.id)"
                              v-for="account in myAccounts"
                              @change="toggleAccount(book, account.id, $event.target.checked)">
                </f7-list-item>
                <f7-list-item :title="tt('You have no account which can be shared.')" v-if="!myAccounts.length"></f7-list-item>
            </f7-list>

            <f7-list strong inset dividers>
                <f7-list-button :href="`/books/settle_up?bookId=${book.id}`">{{ tt('Settle Up') }}</f7-list-button>
                <f7-list-button v-if="book.isOwner" @click="rename(book)">{{ tt('Rename') }}</f7-list-button>
                <f7-list-button color="red" v-if="book.isOwner" @click="remove(book)">{{ tt('Delete') }}</f7-list-button>
                <f7-list-button color="red" v-if="!book.isOwner" @click="leave(book)">{{ tt('Leave This Book') }}</f7-list-button>
            </f7-list>
        </template>
    </f7-page>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading } from '@/lib/ui/mobile.ts';
import { useSharedBooksPageBase } from '@/views/base/books/SharedBooksPageBase.ts';

import type { BookInfoResponse, BookMemberInfoResponse } from '@/models/book.ts';

import { setCurrentBookId } from '@/lib/userstate.ts';

const props = defineProps<{
    f7router: Router.Router;
}>();

const { tt } = useI18n();
const { showConfirm, showPrompt, showToast, routeBackOnError } = useI18nUIComponents();
const {
    loading,
    myAccounts,
    personalBook,
    sharedBooks,
    isCurrentBook,
    switchBook,
    getBookDetail,
    isAccountShared,
    reloadAll,
    createBook,
    renameBook,
    deleteBook,
    leaveBook,
    addMember,
    removeMember,
    setAccountShared
} = useSharedBooksPageBase();

const loadingError = ref<unknown | null>(null);
const newBookName = ref<string>('');

function runWithLoading<T>(promise: () => Promise<T>, successMessage?: string): Promise<T | void> {
    showLoading();

    return promise().then(result => {
        hideLoading();

        if (successMessage) {
            showToast(successMessage);
        }

        return result;
    }).catch(error => {
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function backToPersonalBookIfCurrent(book: BookInfoResponse): void {
    if (isCurrentBook(book)) {
        setCurrentBookId(null);
        window.location.reload();
    }
}

function create(): void {
    const name = newBookName.value.trim();

    if (!name) {
        return;
    }

    runWithLoading(() => createBook(name), 'You have created a shared book').then(book => {
        if (book) {
            newBookName.value = '';
        }
    });
}

function rename(book: BookInfoResponse): void {
    showPrompt('Book Name', book.name, value => {
        const name = (value || '').trim();

        if (name && name !== book.name) {
            runWithLoading(() => renameBook(book.id, name), 'You have renamed this book');
        }
    });
}

function remove(book: BookInfoResponse): void {
    showConfirm('Are you sure you want to delete this shared book?', () => {
        runWithLoading(() => deleteBook(book.id), 'You have deleted this shared book').then(result => {
            if (result) {
                backToPersonalBookIfCurrent(book);
            }
        });
    });
}

function leave(book: BookInfoResponse): void {
    showConfirm('Are you sure you want to leave this book?', () => {
        runWithLoading(() => leaveBook(book.id), 'You have left this book').then(result => {
            if (result) {
                backToPersonalBookIfCurrent(book);
            }
        });
    });
}

function addBookMember(book: BookInfoResponse): void {
    showPrompt('Username of the user to add', '', value => {
        const username = (value || '').trim();

        if (username) {
            runWithLoading(() => addMember(book, username), 'You have added a member to your book');
        }
    });
}

function removeBookMember(book: BookInfoResponse, member: BookMemberInfoResponse): void {
    showConfirm('Are you sure you want to remove this member from your book?', () => {
        runWithLoading(() => removeMember(book, member), 'You have removed this member from your book');
    });
}

function toggleAccount(book: BookInfoResponse, accountId: string, shared: boolean): void {
    runWithLoading(() => setAccountShared(book, accountId, shared));
}

function onPageAfterIn(): void {
    routeBackOnError(props.f7router, loadingError);
}

reloadAll().catch(error => {
    if (!error.processed) {
        loadingError.value = error;
        showToast(error.message || error);
    }
});
</script>
