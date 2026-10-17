<template>
    <f7-page @page:afterin="onPageAfterIn">
        <f7-navbar :title="tt('Shared Books')" :back-link="tt('Back')"></f7-navbar>

        <f7-block-title class="margin-top">{{ tt('Switch Book') }}</f7-block-title>
        <f7-list strong inset dividers :class="{ 'skeleton-text': loading }">
            <f7-list-item radio
                          :key="book.id"
                          :title="getBookDisplayName(book)"
                          :checked="isCurrentBook(book)"
                          v-for="book in allBooks"
                          @change="switchBook(book)">
            </f7-list-item>
        </f7-list>

        <f7-block-title>{{ isInSharedBook ? tt('Book Members') : tt('Share My Book') }}</f7-block-title>
        <f7-block class="no-margin-top margin-bottom-half">
            <small v-if="!isInSharedBook">{{ tt('Members of your book can view and record transactions in it, for example a shared family budget. Each member keeps their own login and their own book.') }}</small>
            <small v-else-if="isInSharedBook">{{ tt('You are working in a book shared with you. Only its owner can add or remove members.') }}</small>
        </f7-block>
        <f7-list strong inset dividers :class="{ 'skeleton-text': loading }">
            <f7-list-item swipeout
                          :id="getMemberDomId(member)"
                          :key="member.id"
                          :title="member.nickname"
                          :footer="member.username"
                          :after="member.isOwner ? tt('Owner') : tt('Member')"
                          v-for="member in members">
                <f7-swipeout-actions :left="textDirection === TextDirection.RTL"
                                     :right="textDirection === TextDirection.LTR"
                                     v-if="!isInSharedBook && !member.isOwner">
                    <f7-swipeout-button color="red" :text="tt('Remove')" @click="removeMember(member)"></f7-swipeout-button>
                </f7-swipeout-actions>
            </f7-list-item>
        </f7-list>

        <f7-list strong inset dividers v-if="!isInSharedBook">
            <f7-list-input
                type="text"
                autocomplete="off"
                clear-button
                :label="tt('Username')"
                :placeholder="tt('Username of the user to add')"
                v-model:value="newMemberUsername"
            ></f7-list-input>
            <f7-list-button :class="{ 'disabled': !newMemberUsername || adding }" @click="addMember">{{ tt('Add Member') }}</f7-list-button>
        </f7-list>

        <f7-list strong inset dividers v-if="isInSharedBook && currentBook">
            <f7-list-button color="red" :class="{ 'disabled': leaving }" @click="leave(currentBook)">{{ tt('Leave This Book') }}</f7-list-button>
        </f7-list>
    </f7-page>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading, onSwipeoutDeleted } from '@/lib/ui/mobile.ts';
import { useBookBase } from '@/views/base/books/BookBase.ts';

import { useBooksStore } from '@/stores/book.ts';

import { TextDirection } from '@/core/text.ts';
import type { BookInfoResponse, BookMemberInfoResponse } from '@/models/book.ts';

import { setCurrentBookId } from '@/lib/userstate.ts';

const props = defineProps<{
    f7router: Router.Router;
}>();

const { tt, getCurrentLanguageTextDirection } = useI18n();
const { showConfirm, showToast, routeBackOnError } = useI18nUIComponents();
const {
    allBooks,
    currentBook,
    isInSharedBook,
    getBookDisplayName,
    isCurrentBook,
    switchBook
} = useBookBase();

const booksStore = useBooksStore();

const loading = ref<boolean>(true);
const loadingError = ref<unknown | null>(null);
const adding = ref<boolean>(false);
const leaving = ref<boolean>(false);
const newMemberUsername = ref<string>('');

const textDirection = computed<TextDirection>(() => getCurrentLanguageTextDirection());
const members = computed<BookMemberInfoResponse[]>(() => booksStore.currentBookMembers);

function getMemberDomId(member: BookMemberInfoResponse): string {
    return 'book_member_' + member.id;
}

function init(): void {
    loading.value = true;

    Promise.all([
        booksStore.loadAllBooks(),
        booksStore.loadCurrentBookMembers()
    ]).then(() => {
        loading.value = false;
    }).catch(error => {
        if (error.processed) {
            loading.value = false;
        } else {
            loadingError.value = error;
            showToast(error.message || error);
        }
    });
}

function addMember(): void {
    const username = newMemberUsername.value.trim();

    if (!username) {
        return;
    }

    adding.value = true;
    showLoading();

    booksStore.addBookMember(username).then(() => {
        adding.value = false;
        hideLoading();
        newMemberUsername.value = '';
        showToast('You have added a member to your book');
    }).catch(error => {
        adding.value = false;
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function removeMember(member: BookMemberInfoResponse): void {
    showConfirm('Are you sure you want to remove this member from your book?', () => {
        showLoading();

        booksStore.removeBookMember(member.id).then(() => {
            hideLoading();
            onSwipeoutDeleted(getMemberDomId(member), () => {
                showToast('You have removed this member from your book');
            });
        }).catch(error => {
            hideLoading();

            if (!error.processed) {
                showToast(error.message || error);
            }
        });
    });
}

function leave(book: BookInfoResponse): void {
    showConfirm('Are you sure you want to leave this book?', () => {
        leaving.value = true;
        showLoading();

        booksStore.leaveBook(book.id).then(() => {
            hideLoading();
            setCurrentBookId(null);
            window.location.reload();
        }).catch(error => {
            leaving.value = false;
            hideLoading();

            if (!error.processed) {
                showToast(error.message || error);
            }
        });
    });
}

function onPageAfterIn(): void {
    routeBackOnError(props.f7router, loadingError);
}

init();
</script>
