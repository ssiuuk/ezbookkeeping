<template>
    <main-page-layout no-navbar>
        <template #content>
            <v-row align-content="start">
                <v-col cols="12">
                    <v-card :class="{ 'disabled': loading }">
                        <template #title>
                            <div class="d-flex align-center flex-wrap">
                                <span>{{ tt('Merchant Report') }}</span>
                                <v-btn-toggle class="ms-3" density="compact" variant="outlined" mandatory divided
                                              :disabled="loading" :model-value="periodType"
                                              @update:model-value="setPeriod">
                                    <v-btn value="month">{{ tt('Monthly') }}</v-btn>
                                    <v-btn value="year">{{ tt('Yearly') }}</v-btn>
                                </v-btn-toggle>
                                <v-spacer/>
                                <div class="d-flex align-center">
                                    <v-btn density="comfortable" variant="text" :icon="mdiChevronLeft"
                                           :aria-label="tt('Previous')" :disabled="loading" @click="changePeriodBy(-1)"/>
                                    <span class="text-body-large mx-2">{{ periodDisplayName }}</span>
                                    <v-btn density="comfortable" variant="text" :icon="mdiChevronRight"
                                           :aria-label="tt('Next')" :disabled="loading" @click="changePeriodBy(1)"/>
                                </div>
                                <v-progress-circular indeterminate size="20" class="ms-3" v-if="loading"></v-progress-circular>
                            </div>
                        </template>

                        <v-card-text>
                            <v-row>
                                <v-col cols="12" sm="4">
                                    <div class="text-body-medium text-medium-emphasis">{{ tt('Total Expense') }}</div>
                                    <div class="text-h6 text-expense">{{ formatAmount(report.totalExpense) }}<span v-if="report.incomplete">+</span></div>
                                </v-col>
                                <v-col cols="12" sm="4">
                                    <div class="text-body-medium text-medium-emphasis">{{ tt('Total Income') }}</div>
                                    <div class="text-h6 text-income">{{ formatAmount(report.totalIncome) }}<span v-if="report.incomplete">+</span></div>
                                </v-col>
                                <v-col cols="12" sm="4">
                                    <div class="text-body-medium text-medium-emphasis">{{ tt('Transactions') }}</div>
                                    <div class="text-h6">{{ report.totalCount }}</div>
                                </v-col>
                                <v-col cols="12" class="pt-0 text-body-medium text-medium-emphasis" v-if="report.incomplete">
                                    {{ tt('Some amounts are left out because their currency has no exchange rate.') }}
                                </v-col>
                            </v-row>
                        </v-card-text>

                        <v-table class="text-no-wrap">
                            <thead>
                            <tr>
                                <th>{{ tt('Merchant') }}</th>
                                <th class="text-right">{{ tt('Expense') }}</th>
                                <th class="merchant-percent-column"></th>
                                <th class="text-right">{{ tt('Income') }}</th>
                                <th class="text-right">{{ tt('Transactions') }}</th>
                            </tr>
                            </thead>
                            <tbody>
                            <tr v-if="!loading && !report.items.length">
                                <td colspan="5">{{ tt('No data') }}</td>
                            </tr>
                            <tr :key="item.merchant" v-for="item in report.items">
                                <td :class="{ 'text-medium-emphasis': !item.merchant }">{{ getMerchantDisplayName(item.merchant) }}</td>
                                <td class="text-right">{{ formatAmount(item.expense) }}</td>
                                <td>
                                    <div class="d-flex align-center">
                                        <v-progress-linear rounded height="6" color="expense" :model-value="item.expensePercent"/>
                                        <span class="ms-2 text-body-small">{{ formatPercent(item.expensePercent) }}</span>
                                    </div>
                                </td>
                                <td class="text-right">{{ formatAmount(item.income) }}</td>
                                <td class="text-right">{{ item.count }}</td>
                            </tr>
                            </tbody>
                        </v-table>
                    </v-card>
                </v-col>
            </v-row>
        </template>
    </main-page-layout>

    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import SnackBar from '@/components/desktop/SnackBar.vue';

import { useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { type MerchantReportPeriodType, useMerchantReportPageBase } from '@/views/base/merchants/MerchantReportPageBase.ts';

import {
    mdiChevronLeft,
    mdiChevronRight
} from '@mdi/js';

type SnackBarType = InstanceType<typeof SnackBar>;

const { tt } = useI18n();
const {
    loading,
    periodType,
    periodDisplayName,
    report,
    loadReport,
    setPeriodType,
    changePeriod,
    getMerchantDisplayName,
    formatAmount,
    formatPercent
} = useMerchantReportPageBase();

const snackbar = useTemplateRef<SnackBarType>('snackbar');

function onError(error: { processed?: boolean, message?: string }): void {
    if (!error.processed) {
        snackbar.value?.showError(error as { message: string });
    }
}

function setPeriod(type: MerchantReportPeriodType): void {
    setPeriodType(type).catch(onError);
}

function changePeriodBy(offset: number): void {
    changePeriod(offset).catch(onError);
}

loadReport().catch(onError);
</script>

<style>
.merchant-percent-column {
    width: 30%;
    min-width: 140px;
}
</style>
