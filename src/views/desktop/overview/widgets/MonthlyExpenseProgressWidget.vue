<template>
    <v-card class="overview-widget expense-progress-widget h-100" :class="{ disabled: loading }">
        <template #title>
            <overview-widget-header :title="title || tt('Monthly Expense Progress')" :icon="mdiCalendarClockOutline" />
        </template>

        <v-card-text class="overview-widget__body">
            <div class="expense-progress-widget__amount text-truncate">
                <span class="overview-widget__amount text-headline-small"
                      :class="{ 'text-expense': !!currentDisplayExpenseAmount, 'text-medium-emphasis': !currentDisplayExpenseAmount }"
                      v-if="!loading || currentDisplayExpenseAmount">{{ currentDisplayExpenseAmount !== '' ? currentDisplayExpenseAmount : tt('No data') }}</span>
                <v-skeleton-loader class="skeleton-no-margin mt-h1 pt-2 mb-2" type="text" width="120px" :loading="true" v-else-if="loading && !currentDisplayExpenseAmount"></v-skeleton-loader>
            </div>
            <div class="overview-widget__caption d-flex justify-space-between mt-2">
                <span>{{ tt('Month elapsed') }}</span>
                <span class="font-weight-medium">{{ displayElapsedPercent }}</span>
            </div>
            <v-progress-linear class="mt-2" color="primary" rounded height="6" :model-value="currentMonthElapsedPercent * 100" />
            <template v-if="budgetProgress">
                <div class="overview-widget__caption d-flex justify-space-between mt-3">
                    <span class="text-truncate">{{ tt('Budget used') }}</span>
                    <span class="font-weight-medium text-truncate ms-2">{{ displayBudgetUsedPercent }} / {{ displayBudgetAmount }}</span>
                </div>
                <v-progress-linear class="mt-2" rounded height="6"
                                   :color="budgetProgressColor"
                                   :model-value="Math.min(budgetProgress.usedRate, 1) * 100" />
            </template>
            <div class="expense-progress-widget__projection mt-3 pt-3">
                <div class="overview-widget__detail-row">
                    <span class="text-truncate">{{ tt('Estimated month-end expense') }}</span>
                    <span class="overview-widget__amount font-weight-medium text-truncate"
                          :class="{ 'text-error': budgetProgress?.projectedOverBudget }"
                          v-if="!loading || currentDisplayExpenseAmount">{{ displayEstimatedExpense }}</span>
                    <v-skeleton-loader class="skeleton-no-margin" type="text" width="100px" :loading="true" v-else-if="loading && !currentDisplayExpenseAmount"></v-skeleton-loader>
                </div>
                <template v-if="budgetProgress && (!loading || currentDisplayExpenseAmount)">
                    <div class="overview-widget__detail-row mt-2">
                        <span class="text-truncate">{{ budgetProgress.overBudget ? tt('Over budget') : tt('Remaining budget') }}</span>
                        <span class="overview-widget__amount font-weight-medium text-truncate"
                              :class="{ 'text-error': budgetProgress.overBudget }">{{ displayBudgetRemainingAmount }}</span>
                    </div>
                    <div class="overview-widget__detail-row overview-widget__caption mt-2" v-if="!budgetProgress.overBudget">
                        <span class="text-truncate">{{ tt('Daily allowance') }}</span>
                        <span class="overview-widget__amount text-truncate">{{ displayDailyAllowance }}</span>
                    </div>
                </template>
                <div class="overview-widget__detail-row overview-widget__caption mt-2">
                    <span class="text-truncate">{{ tt('Last month total') }}</span>
                    <span class="overview-widget__amount text-truncate" v-if="!loading || currentDisplayExpenseAmount">{{ displayLastMonthExpense }}</span>
                    <v-skeleton-loader class="skeleton-no-margin" type="text" width="100px" :loading="true" v-else-if="loading && !currentDisplayExpenseAmount"></v-skeleton-loader>
                </div>
            </div>
        </v-card-text>
    </v-card>
</template>

<script setup lang="ts">
import OverviewWidgetHeader from './OverviewWidgetHeader.vue';

import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useMonthlyExpenseProgressWidgetBase } from '@/views/base/overview/MonthlyExpenseProgressWidgetBase.ts';

import {
    mdiCalendarClockOutline
} from '@mdi/js';

const props = defineProps<{
    loading?: boolean;
    editing?: boolean;
    title?: string;
    budgetAmount?: number;
}>();

const { tt } = useI18n();
const {
    currentDisplayExpenseAmount,
    currentMonthElapsedPercent,
    displayElapsedPercent,
    displayEstimatedExpense,
    displayLastMonthExpense,
    budgetProgress,
    displayBudgetAmount,
    displayBudgetUsedPercent,
    displayBudgetRemainingAmount,
    displayDailyAllowance
} = useMonthlyExpenseProgressWidgetBase(props);

const budgetProgressColor = computed<string>(() => {
    if (budgetProgress.value?.overBudget) {
        return 'error';
    } else if (budgetProgress.value?.aheadOfSchedule) {
        return 'warning';
    } else {
        return 'success';
    }
});
</script>
