<template>
    <f7-list strong inset dividers class="overview-transaction-list no-margin-top margin-bottom">
        <f7-list-item group-title v-if="showTitle">
            <small>{{ title || tt('Monthly Expense Progress') }}</small>
        </f7-list-item>
        <f7-list-item class="item-title-full-line">
            <template #title>
                <div class="overview-widget-primary-amount skeleton-text" v-if="loading">0.00 USD</div>
                <div class="overview-widget-primary-amount" :class="{ 'text-expense': !!currentDisplayExpenseAmount }" v-else-if="!loading">
                    {{ currentDisplayExpenseAmount || tt('No data') }}
                </div>
                <div class="display-flex justify-content-space-between">
                    <small>{{ tt('Month elapsed') }}</small>
                    <small class="margin-left-half text-truncate skeleton-text" v-if="loading">100.0</small>
                    <small v-else-if="!loading">{{ displayElapsedPercent }}</small>
                </div>
                <f7-progressbar class="margin-top-half" :progress="currentMonthElapsedPercent * 100"></f7-progressbar>
                <template v-if="budgetProgress">
                    <div class="display-flex justify-content-space-between margin-top-half">
                        <small>{{ tt('Budget used') }}</small>
                        <small class="margin-left-half text-truncate skeleton-text" v-if="loading">100.0</small>
                        <small class="margin-left-half text-truncate" v-else-if="!loading">{{ displayBudgetUsedPercent }} / {{ displayBudgetAmount }}</small>
                    </div>
                    <f7-progressbar class="margin-top-half"
                                    :color="budgetProgressColor"
                                    :progress="Math.min(budgetProgress.usedRate, 1) * 100"></f7-progressbar>
                </template>
            </template>
        </f7-list-item>
        <f7-list-item class="item-title-full-line">
            <template #footer>
                <div class="overview-transaction-footer default-text-color display-flex justify-content-space-between">
                    <span>{{ tt('Estimated month-end expense') }}</span>
                    <div class="margin-left-half text-truncate skeleton-text" v-if="loading">0.00 USD</div>
                    <span class="margin-left-half text-truncate"
                          :class="{ 'text-color-red': budgetProgress?.projectedOverBudget }"
                          v-else-if="!loading">{{ displayEstimatedExpense }}</span>
                </div>
                <template v-if="budgetProgress">
                    <div class="overview-transaction-footer default-text-color display-flex justify-content-space-between">
                        <span>{{ budgetProgress.overBudget ? tt('Over budget') : tt('Remaining budget') }}</span>
                        <div class="margin-left-half text-truncate skeleton-text" v-if="loading">0.00 USD</div>
                        <span class="margin-left-half text-truncate"
                              :class="{ 'text-color-red': budgetProgress.overBudget }"
                              v-else-if="!loading">{{ displayBudgetRemainingAmount }}</span>
                    </div>
                    <div class="overview-transaction-footer display-flex justify-content-space-between" v-if="!budgetProgress.overBudget">
                        <span>{{ tt('Daily allowance') }}</span>
                        <div class="margin-left-half text-truncate skeleton-text" v-if="loading">0.00 USD</div>
                        <span class="margin-left-half text-truncate" v-else-if="!loading">{{ displayDailyAllowance }}</span>
                    </div>
                </template>
                <div class="overview-transaction-footer display-flex justify-content-space-between">
                    <span>{{ tt('Last month total') }}</span>
                    <div class="margin-left-half text-truncate skeleton-text" v-if="loading">0.00 USD</div>
                    <span class="margin-left-half text-truncate" v-else-if="!loading">{{ displayLastMonthExpense }}</span>
                </div>
            </template>
        </f7-list-item>
    </f7-list>
</template>

<script setup lang="ts">
import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useMonthlyExpenseProgressWidgetBase } from '@/views/base/overview/MonthlyExpenseProgressWidgetBase.ts';

const props = defineProps<{
    loading: boolean;
    title?: string;
    showTitle: boolean;
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
        return 'red';
    } else if (budgetProgress.value?.aheadOfSchedule) {
        return 'orange';
    } else {
        return 'green';
    }
});
</script>
