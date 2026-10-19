<template>
    <f7-page @page:afterin="onPageAfterIn">
        <f7-navbar>
            <f7-nav-left :back-link="tt('Back')"></f7-nav-left>
            <f7-nav-title :title="tt('Budgets')"></f7-nav-title>
            <f7-nav-right :class="{ 'navbar-compact-icons': true, 'disabled': loading || saving }">
                <f7-link icon-f7="pencil" :aria-label="tt('Set Budgets')" @click="startEditing" v-if="!editing"></f7-link>
                <f7-link icon-f7="xmark" :aria-label="tt('Cancel')" @click="cancelEditing" v-if="editing"></f7-link>
                <f7-link icon-f7="checkmark_alt" :aria-label="tt('Save')" @click="save" v-if="editing"></f7-link>
            </f7-nav-right>
        </f7-navbar>

        <f7-toolbar class="budget-month-toolbar" top>
            <f7-link icon-f7="chevron_left" :class="{ 'disabled': loading || saving }" :aria-label="tt('Previous')" @click="changeMonth(-1)"></f7-link>
            <span>{{ currentMonthDisplayName }}</span>
            <f7-link icon-f7="chevron_right" :class="{ 'disabled': loading || saving }" :aria-label="tt('Next')" @click="changeMonth(1)"></f7-link>
        </f7-toolbar>

        <budget-alert-notice />

        <f7-card :class="{ 'skeleton-text': loading }">
            <f7-card-content>
                <div class="budget-summary-row">
                    <span>{{ tt('Total Budget') }}</span>
                    <span>{{ formatAmount(summary.totalBudget) }}<span v-if="incompleteBudget">+</span></span>
                </div>
                <div class="budget-summary-row">
                    <span>{{ tt('Spent') }}</span>
                    <span>{{ formatAmount(summary.totalSpent) }}<span v-if="incompleteSpending">+</span></span>
                </div>
                <div class="budget-summary-row" :class="{ 'text-color-red': summary.totalOverBudget }">
                    <span>{{ summary.totalOverBudget ? tt('Overspent') : tt('Remaining') }}</span>
                    <span>{{ formatAmount(Math.abs(summary.totalRemaining)) }}</span>
                </div>
                <f7-progressbar class="margin-top-half" :color="summary.totalOverBudget ? 'red' : undefined"
                                :progress="Math.min(summary.totalUsedRate, 100)" v-if="summary.totalBudget > 0"></f7-progressbar>
                <div class="margin-top-half text-color-gray" v-if="summary.unbudgetedSpent > 0">
                    <small>{{ tt('format.misc.spentWithoutBudget', { amount: formatAmount(summary.unbudgetedSpent) }) }}</small>
                </div>
            </f7-card-content>
        </f7-card>

        <f7-list strong inset dividers media-list :class="{ 'skeleton-text': loading }">
            <f7-list-item :title="tt('No available category')" v-if="!loading && !summary.rows.length"></f7-list-item>
            <f7-list-item :key="row.category.id"
                          :link="editing ? '#' : undefined"
                          :no-chevron="true"
                          :title="row.category.name"
                          :after="editing ? (editingBudgets[row.category.id] ? formatAmount(editingBudgets[row.category.id] ?? 0, getBudgetCurrency(row.category.id)) : tt('Not set')) : getBudgetDisplayText(row.category.id, row.budget)"
                          v-for="row in summary.rows"
                          @click="editing ? openBudgetSheet(row.category.id) : undefined">
                <template #media>
                    <ItemIcon :icon-type="getCategoryIconType(row.category.iconType)" :icon-id="row.category.icon" :color="row.category.color"></ItemIcon>
                </template>
                <template #text>
                    <div :class="{ 'text-color-red': row.overBudget }">
                        <span>{{ tt('Spent') }} {{ formatAmount(row.spent) }}</span>
                        <span v-if="row.budget"> · {{ row.overBudget ? tt('Overspent') : tt('Remaining') }} {{ formatAmount(Math.abs(row.remaining)) }}</span>
                    </div>
                    <f7-progressbar class="margin-top-half" :color="row.overBudget ? 'red' : undefined"
                                    :progress="Math.min(row.usedRate, 100)" v-if="row.budget && !editing"></f7-progressbar>
                </template>
            </f7-list-item>
        </f7-list>

        <number-pad-sheet :min-value="0"
                          :max-value="TRANSACTION_MAX_AMOUNT"
                          :currency="getBudgetCurrency(editingCategoryId)"
                          v-model:show="showBudgetSheet"
                          v-model="editingAmount"
                          @update:model-value="onBudgetAmountChanged"
        ></number-pad-sheet>
    </f7-page>
</template>

<script setup lang="ts">
import BudgetAlertNotice from '@/components/mobile/BudgetAlertNotice.vue';

import { ref } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading } from '@/lib/ui/mobile.ts';
import { useCategoryBudgetPageBase } from '@/views/base/budgets/CategoryBudgetPageBase.ts';

import { TRANSACTION_MAX_AMOUNT } from '@/consts/transaction.ts';

import { getCategoryIconType } from '@/lib/icon.ts';

const props = defineProps<{
    f7router: Router.Router;
}>();

const { tt } = useI18n();
const { showToast, routeBackOnError } = useI18nUIComponents();
const {
    loading,
    saving,
    editing,
    editingBudgets,
    currentMonthDisplayName,
    summary,
    incompleteSpending,
    incompleteBudget,
    loadAll,
    changeMonth: changeBudgetMonth,
    startEditing,
    cancelEditing,
    saveBudgets,
    getBudgetCurrency,
    getOriginalBudgetText,
    formatAmount
} = useCategoryBudgetPageBase();

const loadingError = ref<unknown | null>(null);
const showBudgetSheet = ref<boolean>(false);
const editingCategoryId = ref<string>('');
const editingAmount = ref<number>(0);

function getBudgetDisplayText(categoryId: string, budget: number): string {
    const originalBudgetText = getOriginalBudgetText(categoryId);

    if (!budget) {
        return originalBudgetText;
    }

    return originalBudgetText ? `${formatAmount(budget)} (${originalBudgetText})` : formatAmount(budget);
}

function openBudgetSheet(categoryId: string): void {
    editingCategoryId.value = categoryId;
    editingAmount.value = editingBudgets.value[categoryId] ?? 0;
    showBudgetSheet.value = true;
}

function onBudgetAmountChanged(amount: number): void {
    if (editingCategoryId.value) {
        editingBudgets.value[editingCategoryId.value] = amount;
    }
}

function changeMonth(offset: number): void {
    changeBudgetMonth(offset).catch(error => {
        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function save(): void {
    showLoading();

    saveBudgets().then(() => {
        hideLoading();
        showToast('You have saved the budgets');
    }).catch(error => {
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function onPageAfterIn(): void {
    routeBackOnError(props.f7router, loadingError);
}

loadAll().catch(error => {
    if (error.processed) {
        return;
    }

    loadingError.value = error;
    showToast(error.message || error);
});
</script>

<style>
.budget-month-toolbar .toolbar-inner {
    justify-content: space-between;
}

.budget-summary-row {
    display: flex;
    justify-content: space-between;
    line-height: 1.8;
}
</style>
