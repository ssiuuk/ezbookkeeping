<template>
    <main-page-layout no-navbar>
        <template #content>
            <v-row align-content="start">
                <v-col cols="12">
                    <v-card>
                        <template #title>
                            <div class="d-flex align-center flex-wrap">
                                <span>{{ tt('Household Report') }}</span>
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
                            <div class="text-body-medium text-medium-emphasis mb-2">{{ tt('Choose the books to include. Your own book holds your personal records, and each shared book holds the records of its members.') }}</div>
                            <div class="d-flex flex-wrap mb-4" v-if="availableBooks.length > 1">
                                <v-checkbox density="compact" hide-details class="me-4 flex-grow-0"
                                            :key="book.id" :label="book.name" :model-value="book.included"
                                            :disabled="loading"
                                            @update:model-value="setBookIncluded(book.id, !!$event)"
                                            v-for="book in availableBooks" />
                            </div>
                            <v-row>
                                <v-col cols="6" md="">
                                    <div class="text-body-medium text-medium-emphasis">{{ tt('Income') }}</div>
                                    <div class="text-h6 text-income">{{ formatAmount(report.totalIncome) }}</div>
                                </v-col>
                                <v-col cols="6" md="">
                                    <div class="text-body-medium text-medium-emphasis">{{ tt('Expense') }}</div>
                                    <div class="text-h6 text-expense">{{ formatAmount(report.totalExpense) }}</div>
                                </v-col>
                                <v-col cols="6" md="">
                                    <div class="text-body-medium text-medium-emphasis">{{ tt('Net Cash Flow') }}</div>
                                    <div class="text-h6" :class="{ 'text-error': report.totalNet < 0 }">{{ formatAmount(report.totalNet) }}</div>
                                </v-col>
                                <v-col cols="6" md="">
                                    <div class="text-body-medium text-medium-emphasis">{{ tt('Savings Rate') }}</div>
                                    <div class="text-h6">{{ formatSavingsRate(report.savingsRate) }}</div>
                                </v-col>
                                <v-col cols="6" md="">
                                    <div class="text-body-medium text-medium-emphasis">{{ tt('Net Worth') }}</div>
                                    <div class="text-h6">{{ formatAmount(report.totalNetWorth) }}</div>
                                </v-col>
                            </v-row>
                            <div class="text-body-small text-medium-emphasis mt-2" v-if="report.incomplete">
                                {{ tt('Some amounts are left out because their currency has no exchange rate.') }}
                            </div>
                        </v-card-text>

                        <v-table class="text-no-wrap">
                            <thead>
                            <tr>
                                <th>{{ tt('Book') }}</th>
                                <th class="text-right">{{ tt('Income') }}</th>
                                <th class="text-right">{{ tt('Expense') }}</th>
                                <th class="text-right">{{ tt('Net Cash Flow') }}</th>
                                <th class="text-right">{{ tt('Net Worth') }}</th>
                            </tr>
                            </thead>
                            <tbody>
                            <tr :key="book.bookId" v-for="book in report.books">
                                <td>{{ book.bookName }}</td>
                                <td class="text-right">{{ formatAmount(book.income) }}</td>
                                <td class="text-right">{{ formatAmount(book.expense) }}</td>
                                <td class="text-right" :class="{ 'text-error': book.net < 0 }">{{ formatAmount(book.net) }}</td>
                                <td class="text-right">{{ formatAmount(book.netWorth) }}</td>
                            </tr>
                            </tbody>
                        </v-table>
                    </v-card>
                </v-col>

                <v-col cols="12">
                    <v-card :title="tt('Expense by Category')">
                        <v-card-text>
                            <div class="text-body-medium text-medium-emphasis" v-if="!loading && !report.expenseByCategory.length">{{ tt('No data') }}</div>
                            <div class="mb-3" :key="item.name" v-for="item in report.expenseByCategory">
                                <div class="d-flex justify-space-between text-body-medium">
                                    <span>{{ item.name }}</span>
                                    <span>{{ formatAmount(item.amount) }}</span>
                                </div>
                                <v-progress-linear rounded height="6" color="expense" :model-value="item.percent"/>
                            </div>
                        </v-card-text>
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
import { type HouseholdReportPeriodType, useHouseholdReportPageBase } from '@/views/base/reports/HouseholdReportPageBase.ts';

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
    availableBooks,
    report,
    loadReport,
    setBookIncluded,
    setPeriodType,
    changePeriod,
    formatAmount,
    formatSavingsRate
} = useHouseholdReportPageBase();

const snackbar = useTemplateRef<SnackBarType>('snackbar');

function onError(error: { processed?: boolean, message?: string }): void {
    if (!error.processed) {
        snackbar.value?.showError(error as { message: string });
    }
}

function setPeriod(type: HouseholdReportPeriodType): void {
    setPeriodType(type).catch(onError);
}

function changePeriodBy(offset: number): void {
    changePeriod(offset).catch(onError);
}

loadReport().catch(onError);
</script>
