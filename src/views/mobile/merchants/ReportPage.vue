<template>
    <f7-page ptr @ptr:refresh="reload" @page:afterin="onPageAfterIn">
        <f7-navbar :title="tt('Merchant Report')" :back-link="tt('Back')"></f7-navbar>

        <f7-block class="margin-vertical-half">
            <f7-segmented strong>
                <f7-button :active="periodType === 'month'" :class="{ 'disabled': loading }" @click="setPeriod('month')">{{ tt('Monthly') }}</f7-button>
                <f7-button :active="periodType === 'year'" :class="{ 'disabled': loading }" @click="setPeriod('year')">{{ tt('Yearly') }}</f7-button>
            </f7-segmented>
        </f7-block>

        <f7-toolbar class="merchant-period-toolbar" top>
            <f7-link icon-f7="chevron_left" :class="{ 'disabled': loading }" :aria-label="tt('Previous')" @click="changePeriodBy(-1)"></f7-link>
            <span>{{ periodDisplayName }}</span>
            <f7-link icon-f7="chevron_right" :class="{ 'disabled': loading }" :aria-label="tt('Next')" @click="changePeriodBy(1)"></f7-link>
        </f7-toolbar>

        <f7-card :class="{ 'skeleton-text': loading }">
            <f7-card-content>
                <div class="merchant-summary-row">
                    <span>{{ tt('Total Expense') }}</span>
                    <span class="text-expense">{{ formatAmount(report.totalExpense) }}</span>
                </div>
                <div class="merchant-summary-row">
                    <span>{{ tt('Total Income') }}</span>
                    <span class="text-income">{{ formatAmount(report.totalIncome) }}</span>
                </div>
                <div class="merchant-summary-row">
                    <span>{{ tt('Transactions') }}</span>
                    <span>{{ report.totalCount }}</span>
                </div>
                <div class="text-color-gray" v-if="report.incomplete">
                    <small>{{ tt('Some amounts are left out because their currency has no exchange rate.') }}</small>
                </div>
            </f7-card-content>
        </f7-card>

        <f7-block-title>{{ tt('Merchant') }}</f7-block-title>
        <f7-list strong inset dividers media-list :class="{ 'skeleton-text': loading }">
            <f7-list-item :title="tt('No data')" v-if="!loading && !report.items.length"></f7-list-item>
            <f7-list-item :key="item.merchant"
                          :class="{ 'merchant-report-no-merchant': !item.merchant }"
                          :title="getMerchantDisplayName(item.merchant)"
                          :after="formatAmount(item.expense)"
                          v-for="item in report.items">
                <template #text>
                    <div>{{ tt('Income') }} {{ formatAmount(item.income) }} · {{ tt('Transactions') }} {{ item.count }}</div>
                    <f7-progressbar class="margin-top-half" :progress="item.expensePercent"></f7-progressbar>
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
import { type MerchantReportPeriodType, useMerchantReportPageBase } from '@/views/base/merchants/MerchantReportPageBase.ts';

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
    getMerchantDisplayName,
    formatAmount
} = useMerchantReportPageBase();

const loadingError = ref<unknown | null>(null);

function onError(error: { processed?: boolean, message?: string }): void {
    if (!error.processed) {
        showToast(error.message || 'Unable to retrieve merchant report');
    }
}

function setPeriod(type: MerchantReportPeriodType): void {
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
.merchant-period-toolbar .toolbar-inner {
    justify-content: space-between;
}

.merchant-summary-row {
    display: flex;
    justify-content: space-between;
    line-height: 1.8;
}

.merchant-report-no-merchant .item-title {
    color: var(--f7-color-gray-tint);
}
</style>
