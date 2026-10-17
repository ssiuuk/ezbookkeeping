<template>
    <f7-page ptr @ptr:refresh="reload" @page:afterin="onPageAfterIn">
        <f7-navbar :title="tt('Household Report')" :back-link="tt('Back')"></f7-navbar>

        <f7-block class="margin-vertical-half">
            <f7-segmented strong>
                <f7-button :active="periodType === 'month'" :class="{ 'disabled': loading }" @click="setPeriod('month')">{{ tt('Monthly') }}</f7-button>
                <f7-button :active="periodType === 'year'" :class="{ 'disabled': loading }" @click="setPeriod('year')">{{ tt('Yearly') }}</f7-button>
            </f7-segmented>
        </f7-block>

        <f7-toolbar class="household-period-toolbar" top>
            <f7-link icon-f7="chevron_left" :class="{ 'disabled': loading }" :aria-label="tt('Previous')" @click="changePeriodBy(-1)"></f7-link>
            <span>{{ periodDisplayName }}</span>
            <f7-link icon-f7="chevron_right" :class="{ 'disabled': loading }" :aria-label="tt('Next')" @click="changePeriodBy(1)"></f7-link>
        </f7-toolbar>

        <f7-card :class="{ 'skeleton-text': loading }">
            <f7-card-content>
                <div class="household-summary-row">
                    <span>{{ tt('Income') }}</span>
                    <span class="text-income">{{ formatAmount(report.totalIncome) }}</span>
                </div>
                <div class="household-summary-row">
                    <span>{{ tt('Expense') }}</span>
                    <span class="text-expense">{{ formatAmount(report.totalExpense) }}</span>
                </div>
                <div class="household-summary-row" :class="{ 'text-color-red': report.totalNet < 0 }">
                    <span>{{ tt('Net Cash Flow') }}</span>
                    <span>{{ formatAmount(report.totalNet) }}</span>
                </div>
                <div class="household-summary-row">
                    <span>{{ tt('Savings Rate') }}</span>
                    <span>{{ formatSavingsRate(report.savingsRate) }}</span>
                </div>
                <div class="household-summary-row">
                    <span>{{ tt('Net Worth') }}</span>
                    <span>{{ formatAmount(report.totalNetWorth) }}</span>
                </div>
                <div class="margin-top-half text-color-gray">
                    <small>{{ tt('Totals of every book you can open, including books shared with you.') }}</small>
                </div>
                <div class="text-color-gray" v-if="report.incomplete">
                    <small>{{ tt('Some amounts are left out because their currency has no exchange rate.') }}</small>
                </div>
            </f7-card-content>
        </f7-card>

        <f7-block-title>{{ tt('Book') }}</f7-block-title>
        <f7-list strong inset dividers media-list :class="{ 'skeleton-text': loading }">
            <f7-list-item :key="book.bookId" :title="book.bookName"
                          :after="formatAmount(book.net)"
                          v-for="book in report.books">
                <template #text>
                    <div>{{ tt('Income') }} {{ formatAmount(book.income) }} · {{ tt('Expense') }} {{ formatAmount(book.expense) }}</div>
                    <div>{{ tt('Net Worth') }} {{ formatAmount(book.netWorth) }}</div>
                </template>
            </f7-list-item>
        </f7-list>

        <f7-block-title>{{ tt('Expense by Category') }}</f7-block-title>
        <f7-list strong inset dividers media-list :class="{ 'skeleton-text': loading }">
            <f7-list-item :title="tt('No data')" v-if="!loading && !report.expenseByCategory.length"></f7-list-item>
            <f7-list-item :key="item.name" :title="item.name" :after="formatAmount(item.amount)"
                          v-for="item in report.expenseByCategory">
                <template #text>
                    <f7-progressbar class="margin-top-half" :progress="item.percent"></f7-progressbar>
                </template>
            </f7-list-item>
        </f7-list>
    </f7-page>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents } from '@/lib/ui/mobile.ts';
import { type HouseholdReportPeriodType, useHouseholdReportPageBase } from '@/views/base/reports/HouseholdReportPageBase.ts';

const props = defineProps<{
    f7router: Router.Router;
}>();

const { tt } = useI18n();
const { showToast, routeBackOnError } = useI18nUIComponents();
const {
    loading,
    periodType,
    periodDisplayName,
    report,
    loadReport,
    setPeriodType,
    changePeriod,
    formatAmount,
    formatSavingsRate
} = useHouseholdReportPageBase();

const loadingError = ref<unknown | null>(null);

function onError(error: { processed?: boolean, message?: string }): void {
    if (!error.processed) {
        showToast(error.message || 'Unable to retrieve transaction statistics');
    }
}

function setPeriod(type: HouseholdReportPeriodType): void {
    setPeriodType(type).catch(onError);
}

function changePeriodBy(offset: number): void {
    changePeriod(offset).catch(onError);
}

function reload(done?: () => void): void {
    loadReport().then(() => done?.()).catch(error => {
        done?.();
        onError(error);
    });
}

function onPageAfterIn(): void {
    routeBackOnError(props.f7router, loadingError);
}

loadReport().catch(error => {
    if (!error.processed) {
        loadingError.value = error;
    }

    onError(error);
});
</script>

<style>
.household-period-toolbar .toolbar-inner {
    justify-content: space-between;
}

.household-summary-row {
    display: flex;
    justify-content: space-between;
    line-height: 1.8;
}
</style>
