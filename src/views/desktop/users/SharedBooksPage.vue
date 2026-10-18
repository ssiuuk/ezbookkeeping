<template>
    <v-row>
        <v-col cols="12">
            <v-card :class="{ 'disabled': loading }">
                <template #title>
                    <span>{{ tt('Shared Books') }}</span>
                    <v-progress-circular indeterminate size="20" class="ms-3" v-if="loading"></v-progress-circular>
                </template>

                <v-card-text class="mt-2">
                    <span class="text-body-large">{{ tt('A shared book sits next to your own book. Add family members to it, and share the accounts they may spend from. Your own book stays private.') }}</span>
                </v-card-text>

                <v-card-text class="pt-0">
                    <v-row>
                        <v-col cols="12" md="6">
                            <v-text-field
                                autocomplete="off"
                                variant="underlined"
                                :disabled="loading || creating"
                                :placeholder="tt('Name of the new shared book, e.g. Family')"
                                v-model="newBookName"
                                @keyup.enter="create"
                            />
                        </v-col>
                        <v-col cols="12" md="6" class="d-flex align-center">
                            <v-btn :disabled="!newBookName.trim() || loading || creating" @click="create">
                                {{ tt('Create Shared Book') }}
                                <v-progress-circular indeterminate size="22" class="ms-2" v-if="creating"></v-progress-circular>
                            </v-btn>
                        </v-col>
                    </v-row>
                </v-card-text>

                <v-table class="text-no-wrap" v-if="personalBook">
                    <tbody>
                    <tr>
                        <td>{{ tt('My Book') }}</td>
                        <td class="text-right">
                            <v-btn density="comfortable" variant="tonal"
                                   :disabled="loading || isCurrentBook(personalBook)"
                                   @click="switchBook(personalBook)">
                                {{ isCurrentBook(personalBook) ? tt('Current Book') : tt('Switch to This Book') }}
                            </v-btn>
                        </td>
                    </tr>
                    </tbody>
                </v-table>
            </v-card>
        </v-col>

        <v-col cols="12" :key="book.id" v-for="book in sharedBooks">
            <v-card :class="{ 'disabled': loading || getBookDetail(book).loading }">
                <template #title>
                    <span>{{ book.name }}</span>
                    <v-chip class="ms-3" size="small" v-if="isCurrentBook(book)">{{ tt('Current Book') }}</v-chip>
                </template>
                <template #subtitle>
                    <span>{{ tt('Owner') }}: {{ book.ownerNickname || book.ownerUsername }}</span>
                </template>
                <template #append>
                    <v-btn density="comfortable" variant="tonal" class="me-2"
                           :disabled="isCurrentBook(book)"
                           @click="switchBook(book)">
                        {{ tt('Switch to This Book') }}
                    </v-btn>
                    <v-btn density="comfortable" color="error" variant="tonal"
                           v-if="book.isOwner" @click="remove(book)">
                        {{ tt('Delete') }}
                    </v-btn>
                    <v-btn density="comfortable" color="error" variant="tonal"
                           v-if="!book.isOwner" @click="leave(book)">
                        {{ tt('Leave') }}
                    </v-btn>
                </template>

                <v-card-text class="pt-2" v-if="book.isOwner">
                    <v-row>
                        <v-col cols="12" md="6">
                            <v-text-field
                                autocomplete="off"
                                variant="underlined"
                                :label="tt('Book Name')"
                                v-model="bookNames[book.id]"
                            />
                        </v-col>
                        <v-col cols="12" md="6" class="d-flex align-center">
                            <v-btn variant="tonal"
                                   :disabled="!bookNames[book.id] || !bookNames[book.id]!.trim() || bookNames[book.id] === book.name"
                                   @click="rename(book)">
                                {{ tt('Rename') }}
                            </v-btn>
                        </v-col>
                    </v-row>
                </v-card-text>

                <v-card-text>
                    <div class="text-title-medium mb-2">{{ tt('Book Members') }}</div>
                    <v-table class="text-no-wrap" density="compact">
                        <tbody>
                        <tr :key="member.id" v-for="member in getBookDetail(book).members">
                            <td>{{ member.nickname || member.username }}</td>
                            <td>{{ member.username }}</td>
                            <td>{{ member.isOwner ? tt('Owner') : tt('Member') }}</td>
                            <td class="text-right">
                                <v-btn density="comfortable" color="error" variant="text"
                                       v-if="book.isOwner && !member.isOwner"
                                       @click="removeBookMember(book, member)">
                                    {{ tt('Remove') }}
                                </v-btn>
                            </td>
                        </tr>
                        </tbody>
                    </v-table>
                    <v-row class="mt-1" v-if="book.isOwner">
                        <v-col cols="12" md="6">
                            <v-text-field
                                autocomplete="off"
                                variant="underlined"
                                :placeholder="tt('Username of the user to add')"
                                v-model="newMemberUsernames[book.id]"
                                @keyup.enter="addBookMember(book)"
                            />
                        </v-col>
                        <v-col cols="12" md="6" class="d-flex align-center">
                            <v-btn variant="tonal" :disabled="!newMemberUsernames[book.id] || !newMemberUsernames[book.id]!.trim()" @click="addBookMember(book)">
                                {{ tt('Add Member') }}
                            </v-btn>
                        </v-col>
                    </v-row>
                </v-card-text>

                <v-card-text>
                    <div class="text-title-medium mb-1">{{ tt('Shared Accounts') }}</div>
                    <div class="text-body-medium mb-2">{{ tt('Members of this book can record transactions with these accounts, and see their balances.') }}</div>
                    <v-table class="text-no-wrap" density="compact" v-if="getBookDetail(book).sharedAccounts.some(account => !account.isMine)">
                        <tbody>
                        <tr :key="account.accountId" v-for="account in getBookDetail(book).sharedAccounts.filter(item => !item.isMine)">
                            <td>{{ account.name }}</td>
                            <td class="text-right">
                                <v-btn density="comfortable" color="error" variant="text"
                                       v-if="book.isOwner"
                                       @click="toggleAccount(book, account.accountId, false)">
                                    {{ tt('Stop Sharing') }}
                                </v-btn>
                            </td>
                        </tr>
                        </tbody>
                    </v-table>
                    <div class="text-body-medium mt-3 mb-1">{{ tt('My accounts to share') }}</div>
                    <div class="text-body-medium" v-if="!myAccounts.length">{{ tt('You have no account which can be shared.') }}</div>
                    <v-switch density="compact" hide-details color="primary"
                              :key="account.id"
                              :label="account.name"
                              :model-value="isAccountShared(book, account.id)"
                              @update:model-value="toggleAccount(book, account.id, !!$event)"
                              v-for="account in myAccounts" />
                </v-card-text>
            </v-card>
        </v-col>
    </v-row>

    <confirm-dialog ref="confirmDialog"/>
    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import SnackBar from '@/components/desktop/SnackBar.vue';

import { ref, watch, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useSharedBooksPageBase } from '@/views/base/books/SharedBooksPageBase.ts';

import type { BookInfoResponse, BookMemberInfoResponse } from '@/models/book.ts';

import { setCurrentBookId } from '@/lib/userstate.ts';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type SnackBarType = InstanceType<typeof SnackBar>;

const { tt } = useI18n();
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

const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const snackbar = useTemplateRef<SnackBarType>('snackbar');

const creating = ref<boolean>(false);
const newBookName = ref<string>('');
const bookNames = ref<Record<string, string>>({});
const newMemberUsernames = ref<Record<string, string>>({});

watch(sharedBooks, books => {
    for (const book of books) {
        if (bookNames.value[book.id] === undefined) {
            bookNames.value[book.id] = book.name;
        }
    }
}, { immediate: true });

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function showError(error: any): void {
    if (!error.processed) {
        snackbar.value?.showError(error);
    }
}

function create(): void {
    const name = newBookName.value.trim();

    if (!name) {
        return;
    }

    creating.value = true;

    createBook(name).then(() => {
        creating.value = false;
        newBookName.value = '';
        snackbar.value?.showMessage('You have created a shared book');
    }).catch(error => {
        creating.value = false;
        showError(error);
    });
}

function rename(book: BookInfoResponse): void {
    const name = (bookNames.value[book.id] || '').trim();

    renameBook(book.id, name).then(() => {
        snackbar.value?.showMessage('You have renamed this book');
    }).catch(showError);
}

function remove(book: BookInfoResponse): void {
    confirmDialog.value?.open('Are you sure you want to delete this shared book?', { color: 'error' }).then(() => {
        deleteBook(book.id).then(() => {
            if (isCurrentBook(book)) {
                setCurrentBookId(null);
                window.location.reload();
                return;
            }

            snackbar.value?.showMessage('You have deleted this shared book');
        }).catch(showError);
    });
}

function leave(book: BookInfoResponse): void {
    confirmDialog.value?.open('Are you sure you want to leave this book?', { color: 'error' }).then(() => {
        leaveBook(book.id).then(() => {
            if (isCurrentBook(book)) {
                setCurrentBookId(null);
                window.location.reload();
                return;
            }

            snackbar.value?.showMessage('You have left this book');
        }).catch(showError);
    });
}

function addBookMember(book: BookInfoResponse): void {
    const username = (newMemberUsernames.value[book.id] || '').trim();

    if (!username) {
        return;
    }

    addMember(book, username).then(() => {
        newMemberUsernames.value[book.id] = '';
        snackbar.value?.showMessage('You have added a member to your book');
    }).catch(showError);
}

function removeBookMember(book: BookInfoResponse, member: BookMemberInfoResponse): void {
    confirmDialog.value?.open('Are you sure you want to remove this member from your book?', { color: 'error' }).then(() => {
        removeMember(book, member).then(() => {
            snackbar.value?.showMessage('You have removed this member from your book');
        }).catch(showError);
    });
}

function toggleAccount(book: BookInfoResponse, accountId: string, shared: boolean): void {
    setAccountShared(book, accountId, shared).catch(showError);
}

reloadAll().catch(showError);
</script>
