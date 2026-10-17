<template>
    <f7-page ptr infinite :infinite-preloader="loadingMore" @ptr:refresh="reload" @infinite="loadMore" @page:afterin="onPageAfterIn">
        <f7-navbar :title="project?.name ?? tt('Project Report')" :back-link="tt('Back')"></f7-navbar>

        <f7-card :class="{ 'skeleton-text': loading }">
            <f7-card-content>
                <div class="project-summary-row">
                    <span>{{ tt('Total Expense') }}</span>
                    <span class="text-expense">{{ formatAmount(report.totalExpense) }}</span>
                </div>
                <div class="project-summary-row">
                    <span>{{ tt('Total Income') }}</span>
                    <span class="text-income">{{ formatAmount(report.totalIncome) }}</span>
                </div>
                <div class="project-summary-row" :class="{ 'text-color-red': report.net < 0 }">
                    <span>{{ tt('Net Amount') }}</span>
                    <span>{{ formatAmount(report.net) }}</span>
                </div>
                <div class="project-summary-row">
                    <span>{{ tt('Date Range') }}</span>
                    <span>{{ dateRangeText }}</span>
                </div>
                <div class="margin-top-half text-color-gray" v-if="project?.transactionCount">
                    <small>{{ tt('format.misc.projectTransactionCount', { count: project.transactionCount }) }}</small>
                </div>
                <div class="text-color-gray" v-if="project?.comment">
                    <small>{{ project.comment }}</small>
                </div>
                <div class="text-color-gray" v-if="report.incomplete">
                    <small>{{ tt('Some amounts are left out because their currency has no exchange rate.') }}</small>
                </div>
            </f7-card-content>
        </f7-card>

        <template :key="section.title" v-for="section in categorySections">
            <f7-block-title>{{ section.title }}</f7-block-title>
            <f7-list strong inset dividers media-list :class="{ 'skeleton-text': loading }">
                <f7-list-item :title="tt('No data')" v-if="!loading && !section.items.length"></f7-list-item>
                <f7-list-item :key="item.categoryId" :title="item.name" :after="formatAmount(item.amount)"
                              v-for="item in section.items">
                    <template #media>
                        <ItemIcon :icon-type="getCategoryIconType(item.iconType)" :icon-id="item.icon" :color="item.color"></ItemIcon>
                    </template>
                    <template #text>
                        <f7-progressbar class="margin-top-half" :progress="item.percent"></f7-progressbar>
                    </template>
                </f7-list-item>
            </f7-list>
        </template>

        <f7-block-title>{{ tt('Project Transactions') }}</f7-block-title>
        <f7-list strong inset dividers media-list :class="{ 'skeleton-text': loading }">
            <f7-list-item :title="tt('No transactions')" v-if="!loading && !transactions.length"></f7-list-item>
            <f7-list-item :key="transaction.id"
                          :link="`/transaction/detail?id=${transaction.id}`"
                          :title="getTransactionTitle(transaction)"
                          :text="getTransactionDescription(transaction)"
                          v-for="transaction in transactions">
                <template #media>
                    <ItemIcon :icon-type="getCategoryIconType(getTransactionCategory(transaction)?.iconType)"
                              :icon-id="getTransactionCategory(transaction)?.icon ?? ''"
                              :color="getTransactionCategory(transaction)?.color"
                              v-if="getTransactionCategory(transaction)"></ItemIcon>
                    <f7-icon f7="arrow_right_arrow_left" v-else></f7-icon>
                </template>
                <template #after>
                    <span :class="getTransactionAmountClass(transaction)">{{ getTransactionAmount(transaction) }}</span>
                </template>
            </f7-list-item>
        </f7-list>
    </f7-page>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents } from '@/lib/ui/mobile.ts';
import { useProjectReportPageBase } from '@/views/base/projects/ProjectReportPageBase.ts';

import type { ProjectCategorySummary } from '@/lib/project_report.ts';

import { getCategoryIconType } from '@/lib/icon.ts';

interface CategorySection {
    readonly title: string;
    readonly items: ProjectCategorySummary[];
}

const props = defineProps<{
    f7route: Router.Route;
    f7router: Router.Router;
}>();

const { tt } = useI18n();
const { showToast, routeBackOnError } = useI18nUIComponents();
const {
    loading,
    loadingMore,
    project,
    transactions,
    hasMoreTransactions,
    dateRangeText,
    report,
    loadAll,
    loadMoreTransactions,
    formatAmount,
    getTransactionCategory,
    getTransactionTitle,
    getTransactionDescription,
    getTransactionAmount,
    getTransactionAmountClass
} = useProjectReportPageBase();

const loadingError = ref<unknown | null>(null);
const projectId = (props.f7route.query['id'] as string | undefined) ?? '';

const categorySections = computed<CategorySection[]>(() => [
    { title: tt('Expense by Category'), items: report.value.expenseByCategory },
    { title: tt('Income by Category'), items: report.value.incomeByCategory }
]);

function load(done?: () => void): void {
    loadAll(projectId).then(() => {
        done?.();
    }).catch(error => {
        done?.();

        if (error.processed) {
            return;
        }

        if (!done) {
            loadingError.value = error;
        }

        showToast(error.message || error);
    });
}

function reload(done?: () => void): void {
    load(done);
}

function loadMore(): void {
    if (loading.value || !hasMoreTransactions.value) {
        return;
    }

    loadMoreTransactions().catch(error => {
        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function onPageAfterIn(): void {
    routeBackOnError(props.f7router, loadingError);
}

if (projectId) {
    load();
} else {
    loadingError.value = 'Parameter Invalid';
    showToast('Parameter Invalid');
}
</script>

<style>
.project-summary-row {
    display: flex;
    justify-content: space-between;
    line-height: 1.8;
}
</style>
