<template>
    <v-row>
        <v-col cols="12">
            <v-card :class="{ 'disabled': loading }">
                <template #title>
                    <span>{{ isInSharedBook ? tt('Book Members') : tt('Share My Book') }}</span>
                    <v-progress-circular indeterminate size="20" class="ms-3" v-if="loading"></v-progress-circular>
                </template>

                <v-card-text class="mt-2">
                    <span class="text-body-large" v-if="!isInSharedBook">
                        {{ tt('Members of your book can view and record transactions in it, for example a shared family budget. Each member keeps their own login and their own book.') }}
                    </span>
                    <span class="text-body-large" v-else-if="isInSharedBook">
                        {{ tt('You are working in a book shared with you. Only its owner can add or remove members.') }}
                    </span>
                </v-card-text>

                <v-card-text class="pt-0" v-if="!isInSharedBook">
                    <v-row>
                        <v-col cols="12" md="6">
                            <v-text-field
                                autocomplete="off"
                                variant="underlined"
                                :disabled="loading || adding"
                                :placeholder="tt('Username of the user to add')"
                                v-model="newMemberUsername"
                                @keyup.enter="addMember"
                            />
                        </v-col>
                        <v-col cols="12" md="6" class="d-flex align-center">
                            <v-btn :disabled="!newMemberUsername || loading || adding" @click="addMember">
                                {{ tt('Add Member') }}
                                <v-progress-circular indeterminate size="22" class="ms-2" v-if="adding"></v-progress-circular>
                            </v-btn>
                        </v-col>
                    </v-row>
                </v-card-text>

                <v-table class="text-no-wrap">
                    <thead>
                    <tr>
                        <th>{{ tt('Nickname') }}</th>
                        <th>{{ tt('Username') }}</th>
                        <th>{{ tt('Role') }}</th>
                        <th class="text-right" v-if="!isInSharedBook">{{ tt('Operation') }}</th>
                    </tr>
                    </thead>
                    <tbody>
                    <tr :key="member.id" v-for="member in members">
                        <td>{{ member.nickname }}</td>
                        <td>{{ member.username }}</td>
                        <td>{{ member.isOwner ? tt('Owner') : tt('Member') }}</td>
                        <td class="text-right" v-if="!isInSharedBook">
                            <v-btn density="comfortable" color="error" variant="tonal"
                                   :disabled="loading || removing[member.id]"
                                   v-if="!member.isOwner"
                                   @click="removeMember(member)">
                                {{ tt('Remove') }}
                            </v-btn>
                        </td>
                    </tr>
                    </tbody>
                </v-table>
            </v-card>
        </v-col>

        <v-col cols="12">
            <v-card :class="{ 'disabled': loading }">
                <template #title>
                    <span>{{ tt('Books Shared with Me') }}</span>
                </template>

                <v-card-text class="mt-2" v-if="!sharedBooks.length">
                    <span class="text-body-large">{{ tt('No book has been shared with you yet.') }}</span>
                </v-card-text>

                <v-table class="text-no-wrap" v-if="sharedBooks.length">
                    <tbody>
                    <tr :key="book.id" v-for="book in sharedBooks">
                        <td>{{ getBookDisplayName(book) }}</td>
                        <td class="text-right">
                            <v-btn density="comfortable" variant="tonal" class="me-2"
                                   :disabled="loading || isCurrentBook(book)"
                                   @click="switchBook(book)">
                                {{ isCurrentBook(book) ? tt('Current Book') : tt('Switch to This Book') }}
                            </v-btn>
                            <v-btn density="comfortable" color="error" variant="tonal"
                                   :disabled="loading || leaving[book.id]"
                                   @click="leave(book)">
                                {{ tt('Leave') }}
                            </v-btn>
                        </td>
                    </tr>
                    </tbody>
                </v-table>
            </v-card>
        </v-col>
    </v-row>

    <confirm-dialog ref="confirmDialog"/>
    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import SnackBar from '@/components/desktop/SnackBar.vue';

import { ref, computed, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useBookBase } from '@/views/base/books/BookBase.ts';

import { useBooksStore } from '@/stores/book.ts';

import type { BookInfoResponse, BookMemberInfoResponse } from '@/models/book.ts';

import { setCurrentBookId } from '@/lib/userstate.ts';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type SnackBarType = InstanceType<typeof SnackBar>;

const { tt } = useI18n();
const {
    allBooks,
    isInSharedBook,
    getBookDisplayName,
    isCurrentBook,
    switchBook
} = useBookBase();

const booksStore = useBooksStore();

const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const snackbar = useTemplateRef<SnackBarType>('snackbar');

const loading = ref<boolean>(true);
const adding = ref<boolean>(false);
const removing = ref<Record<string, boolean>>({});
const leaving = ref<Record<string, boolean>>({});
const newMemberUsername = ref<string>('');

const members = computed<BookMemberInfoResponse[]>(() => booksStore.currentBookMembers);
const sharedBooks = computed<BookInfoResponse[]>(() => allBooks.value.filter(book => !book.isOwner));

function reload(): void {
    loading.value = true;

    Promise.all([
        booksStore.loadAllBooks(),
        booksStore.loadCurrentBookMembers()
    ]).then(() => {
        loading.value = false;
    }).catch(error => {
        loading.value = false;

        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function addMember(): void {
    const username = newMemberUsername.value.trim();

    if (!username) {
        return;
    }

    adding.value = true;

    booksStore.addBookMember(username).then(() => {
        adding.value = false;
        newMemberUsername.value = '';
        snackbar.value?.showMessage('You have added a member to your book');
    }).catch(error => {
        adding.value = false;

        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function removeMember(member: BookMemberInfoResponse): void {
    confirmDialog.value?.open('Are you sure you want to remove this member from your book?', { color: 'error' }).then(() => {
        removing.value[member.id] = true;

        booksStore.removeBookMember(member.id).then(() => {
            removing.value[member.id] = false;
            snackbar.value?.showMessage('You have removed this member from your book');
        }).catch(error => {
            removing.value[member.id] = false;

            if (!error.processed) {
                snackbar.value?.showError(error);
            }
        });
    });
}

function leave(book: BookInfoResponse): void {
    confirmDialog.value?.open('Are you sure you want to leave this book?', { color: 'error' }).then(() => {
        leaving.value[book.id] = true;

        booksStore.leaveBook(book.id).then(() => {
            leaving.value[book.id] = false;

            if (isCurrentBook(book)) {
                setCurrentBookId(null);
                window.location.reload();
                return;
            }

            snackbar.value?.showMessage('You have left this book');
        }).catch(error => {
            leaving.value[book.id] = false;

            if (!error.processed) {
                snackbar.value?.showError(error);
            }
        });
    });
}

reload();
</script>
