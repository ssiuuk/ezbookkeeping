<template>
    <main-page-layout no-navbar>
        <template #content>
            <v-row align-content="start">
                <v-col cols="12">
                    <v-card :class="{ 'disabled': loading }">
                        <template #title>
                            <div class="d-flex align-center flex-wrap">
                                <v-btn density="comfortable" variant="text" :icon="true" :aria-label="tt('Back')" to="/projects">
                                    <v-icon :icon="mdiArrowLeft" />
                                    <v-tooltip activator="parent">{{ tt('Projects') }}</v-tooltip>
                                </v-btn>
                                <span class="ms-2">{{ project?.name ?? tt('Project Report') }}</span>
                                <v-chip class="ms-2" size="small" v-if="project?.hidden">{{ tt('Hidden') }}</v-chip>
                                <v-progress-circular indeterminate size="20" class="ms-3" v-if="loading"></v-progress-circular>
                            </div>
                        </template>

                        <v-card-text class="pt-0" v-if="project?.comment">
                            <span class="text-body-medium text-medium-emphasis">{{ project.comment }}</span>
                        </v-card-text>

                        <v-card-text>
                            <v-row>
                                <v-col cols="12" sm="6" md="3">
                                    <div class="text-body-medium text-medium-emphasis">{{ tt('Total Expense') }}</div>
                                    <div class="text-h6 text-expense">{{ formatAmount(report.totalExpense) }}<span v-if="report.incomplete">+</span></div>
                                </v-col>
                                <v-col cols="12" sm="6" md="3">
                                    <div class="text-body-medium text-medium-emphasis">{{ tt('Total Income') }}</div>
                                    <div class="text-h6 text-income">{{ formatAmount(report.totalIncome) }}<span v-if="report.incomplete">+</span></div>
                                </v-col>
                                <v-col cols="12" sm="6" md="3">
                                    <div class="text-body-medium text-medium-emphasis">{{ tt('Net Amount') }}</div>
                                    <div class="text-h6">{{ formatAmount(report.net) }}</div>
                                </v-col>
                                <v-col cols="12" sm="6" md="3">
                                    <div class="text-body-medium text-medium-emphasis">{{ tt('Date Range') }}</div>
                                    <div class="text-h6">{{ dateRangeText }}</div>
                                    <div class="text-body-small text-medium-emphasis" v-if="project?.transactionCount">{{ tt('format.misc.projectTransactionCount', { count: project.transactionCount }) }}</div>
                                </v-col>
                                <v-col cols="12" class="pt-0 text-body-medium text-medium-emphasis" v-if="report.incomplete">
                                    {{ tt('Some amounts are left out because their currency has no exchange rate.') }}
                                </v-col>
                            </v-row>
                        </v-card-text>
                    </v-card>
                </v-col>

                <v-col cols="12" md="6" :key="section.title" v-for="section in categorySections">
                    <v-card class="h-100" :class="{ 'disabled': loading }">
                        <template #title>
                            <span>{{ section.title }}</span>
                        </template>
                        <v-table class="text-no-wrap">
                            <thead>
                            <tr>
                                <th>{{ tt('Category') }}</th>
                                <th class="text-right">{{ tt('Amount') }}</th>
                                <th class="project-percent-column"></th>
                            </tr>
                            </thead>
                            <tbody>
                            <tr v-if="!loading && !section.items.length">
                                <td colspan="3">{{ tt('No data') }}</td>
                            </tr>
                            <tr :key="item.categoryId" v-for="item in section.items">
                                <td>
                                    <div class="d-flex align-center">
                                        <ItemIcon size="24px" :icon-type="getCategoryIconType(item.iconType)" :icon-id="item.icon" :color="item.color"></ItemIcon>
                                        <span class="ms-3">{{ item.name }}</span>
                                    </div>
                                </td>
                                <td class="text-right">{{ formatAmount(item.amount) }}</td>
                                <td>
                                    <div class="d-flex align-center">
                                        <v-progress-linear rounded height="6" :color="section.color" :model-value="item.percent"/>
                                        <span class="ms-2 text-body-small">{{ formatPercent(item.percent) }}</span>
                                    </div>
                                </td>
                            </tr>
                            </tbody>
                        </v-table>
                    </v-card>
                </v-col>

                <v-col cols="12">
                    <v-card :class="{ 'disabled': loading }">
                        <template #title>
                            <span>{{ tt('Project Transactions') }}</span>
                        </template>
                        <v-list class="py-0" v-if="transactions.length">
                            <v-list-item :key="transaction.id"
                                         :title="getTransactionTitle(transaction)"
                                         :subtitle="getTransactionDescription(transaction)"
                                         v-for="transaction in transactions"
                                         @click="showTransaction(transaction)">
                                <template #prepend>
                                    <ItemIcon class="me-4" size="24px" :icon-type="getCategoryIconType(getTransactionCategory(transaction)?.iconType)"
                                              :icon-id="getTransactionCategory(transaction)?.icon ?? ''"
                                              :color="getTransactionCategory(transaction)?.color"
                                              v-if="getTransactionCategory(transaction)" />
                                    <v-icon class="me-4" size="24" :icon="mdiSwapHorizontal" v-else />
                                </template>
                                <template #append>
                                    <span :class="getTransactionAmountClass(transaction)">{{ getTransactionAmount(transaction) }}</span>
                                </template>
                            </v-list-item>
                        </v-list>
                        <v-card-text v-else-if="!loading">{{ tt('No transactions') }}</v-card-text>
                        <v-card-text class="d-flex justify-center" v-if="hasMoreTransactions">
                            <v-btn variant="tonal" :disabled="loadingMore" @click="loadMore">
                                {{ tt('Load More') }}
                                <v-progress-circular indeterminate size="18" class="ms-2" v-if="loadingMore"></v-progress-circular>
                            </v-btn>
                        </v-card-text>
                    </v-card>
                </v-col>
            </v-row>
        </template>
    </main-page-layout>

    <edit-dialog ref="editDialog" :type="TransactionEditPageType.Transaction" />
    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import SnackBar from '@/components/desktop/SnackBar.vue';
import EditDialog from '@/views/desktop/transactions/list/dialogs/EditDialog.vue';

import { computed, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { TransactionEditPageType } from '@/views/base/transactions/TransactionEditPageBase.ts';
import { useProjectReportPageBase } from '@/views/base/projects/ProjectReportPageBase.ts';

import type { ProjectCategorySummary } from '@/lib/project_report.ts';
import { type TransactionInfoResponse, Transaction } from '@/models/transaction.ts';

import { getCategoryIconType } from '@/lib/icon.ts';

import {
    mdiArrowLeft,
    mdiSwapHorizontal
} from '@mdi/js';

type SnackBarType = InstanceType<typeof SnackBar>;
type EditDialogType = InstanceType<typeof EditDialog>;

interface CategorySection {
    readonly title: string;
    readonly color: string;
    readonly items: ProjectCategorySummary[];
}

const props = defineProps<{
    initId?: string;
}>();

const { tt, formatPercentToLocalizedNumerals } = useI18n();
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

const snackbar = useTemplateRef<SnackBarType>('snackbar');
const editDialog = useTemplateRef<EditDialogType>('editDialog');

const categorySections = computed<CategorySection[]>(() => [
    { title: tt('Expense by Category'), color: 'expense', items: report.value.expenseByCategory },
    { title: tt('Income by Category'), color: 'income', items: report.value.incomeByCategory }
]);

function formatPercent(value: number): string {
    return formatPercentToLocalizedNumerals(value, 1, '<0.1');
}

function showError(error: unknown): void {
    if (!(error as { processed?: boolean }).processed) {
        snackbar.value?.showError(error as { message: string });
    }
}

function reload(): void {
    if (!props.initId) {
        snackbar.value?.showMessage('Parameter Invalid');
        return;
    }

    loadAll(props.initId).catch(showError);
}

function loadMore(): void {
    loadMoreTransactions().catch(showError);
}

function showTransaction(transaction: TransactionInfoResponse): void {
    editDialog.value?.open({
        id: transaction.id,
        currentTransaction: Transaction.of(transaction)
    }).then(result => {
        if (result && result.message) {
            snackbar.value?.showMessage(result.message);
        }

        reload();
    }).catch(error => {
        if (error) {
            snackbar.value?.showError(error);
        }
    });
}

reload();
</script>

<style>
.project-percent-column {
    width: 35%;
    min-width: 140px;
}
</style>
