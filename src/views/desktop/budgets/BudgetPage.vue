<template>
    <v-row>
        <v-col cols="12">
            <v-card>
                <template #title>
                    <div class="d-flex align-center flex-wrap">
                        <span>{{ tt('Budgets') }}</span>
                        <v-btn class="ms-3" density="compact" color="default" variant="outlined"
                               :disabled="loading || saving" @click="editing ? cancelEditing() : startEditing()">
                            {{ editing ? tt('Cancel') : tt('Set Budgets') }}
                        </v-btn>
                        <v-btn class="ms-2" density="compact" color="primary" variant="flat"
                               :disabled="loading || saving" @click="save" v-if="editing">
                            {{ tt('Save') }}
                            <v-progress-circular indeterminate size="18" class="ms-2" v-if="saving"></v-progress-circular>
                        </v-btn>
                        <v-spacer/>
                        <div class="d-flex align-center">
                            <v-btn density="comfortable" variant="text" :icon="mdiChevronLeft"
                                   :aria-label="tt('Previous')" :disabled="loading || saving" @click="changeMonth(-1)"/>
                            <span class="text-body-large mx-2">{{ currentMonthDisplayName }}</span>
                            <v-btn density="comfortable" variant="text" :icon="mdiChevronRight"
                                   :aria-label="tt('Next')" :disabled="loading || saving" @click="changeMonth(1)"/>
                        </div>
                        <v-progress-circular indeterminate size="20" class="ms-3" v-if="loading"></v-progress-circular>
                    </div>
                </template>

                <v-card-text>
                    <v-row>
                        <v-col cols="12" md="4">
                            <div class="text-body-medium text-medium-emphasis">{{ tt('Total Budget') }}</div>
                            <div class="text-h6">{{ formatAmount(summary.totalBudget) }}</div>
                        </v-col>
                        <v-col cols="12" md="4">
                            <div class="text-body-medium text-medium-emphasis">{{ tt('Spent') }}</div>
                            <div class="text-h6">{{ formatAmount(summary.totalSpent) }}<span v-if="incompleteSpending">+</span></div>
                        </v-col>
                        <v-col cols="12" md="4">
                            <div class="text-body-medium text-medium-emphasis">{{ summary.totalOverBudget ? tt('Overspent') : tt('Remaining') }}</div>
                            <div class="text-h6" :class="{ 'text-error': summary.totalOverBudget }">{{ formatAmount(Math.abs(summary.totalRemaining)) }}</div>
                        </v-col>
                        <v-col cols="12" class="pt-0" v-if="summary.totalBudget > 0">
                            <v-progress-linear rounded height="8"
                                               :color="summary.totalOverBudget ? 'error' : 'primary'"
                                               :model-value="Math.min(summary.totalUsedRate, 100)"/>
                        </v-col>
                        <v-col cols="12" class="pt-0 text-body-medium text-medium-emphasis" v-if="summary.unbudgetedSpent > 0">
                            {{ tt('format.misc.spentWithoutBudget', { amount: formatAmount(summary.unbudgetedSpent) }) }}
                        </v-col>
                    </v-row>
                </v-card-text>

                <v-table class="text-no-wrap">
                    <thead>
                    <tr>
                        <th>{{ tt('Category') }}</th>
                        <th class="text-right budget-amount-column">{{ tt('Budget') }}</th>
                        <th class="text-right">{{ tt('Spent') }}</th>
                        <th class="text-right">{{ tt('Remaining') }}</th>
                        <th class="budget-progress-column"></th>
                    </tr>
                    </thead>
                    <tbody>
                    <tr v-if="!loading && !summary.rows.length">
                        <td colspan="5">{{ tt('No available category') }}</td>
                    </tr>
                    <tr :key="row.category.id" v-for="row in summary.rows">
                        <td>
                            <div class="d-flex align-center">
                                <ItemIcon size="24px" :icon-type="getCategoryIconType(row.category.iconType)" :icon-id="row.category.icon" :color="row.category.color"></ItemIcon>
                                <span class="ms-3">{{ row.category.name }}</span>
                            </div>
                        </td>
                        <td class="text-right">
                            <amount-input density="compact" :currency="defaultCurrency" :show-currency="true"
                                          :disabled="saving"
                                          v-model="editingBudgets[row.category.id]"
                                          v-if="editing"/>
                            <span class="text-medium-emphasis" v-else-if="!row.budget">-</span>
                            <span v-else>{{ formatAmount(row.budget) }}</span>
                        </td>
                        <td class="text-right">{{ formatAmount(row.spent) }}</td>
                        <td class="text-right" :class="{ 'text-error': row.overBudget }">
                            <span v-if="row.budget">{{ row.overBudget ? '-' : '' }}{{ formatAmount(Math.abs(row.remaining)) }}</span>
                            <span class="text-medium-emphasis" v-else>-</span>
                        </td>
                        <td>
                            <v-progress-linear rounded height="6"
                                               :color="row.overBudget ? 'error' : 'primary'"
                                               :model-value="Math.min(row.usedRate, 100)"
                                               v-if="row.budget"/>
                        </td>
                    </tr>
                    </tbody>
                </v-table>
            </v-card>
        </v-col>
    </v-row>

    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import SnackBar from '@/components/desktop/SnackBar.vue';

import { useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useCategoryBudgetPageBase } from '@/views/base/budgets/CategoryBudgetPageBase.ts';

import { getCategoryIconType } from '@/lib/icon.ts';

import {
    mdiChevronLeft,
    mdiChevronRight
} from '@mdi/js';

type SnackBarType = InstanceType<typeof SnackBar>;

const { tt } = useI18n();
const {
    loading,
    saving,
    editing,
    editingBudgets,
    defaultCurrency,
    currentMonthDisplayName,
    summary,
    incompleteSpending,
    loadAll,
    changeMonth: changeBudgetMonth,
    startEditing,
    cancelEditing,
    saveBudgets,
    formatAmount
} = useCategoryBudgetPageBase();

const snackbar = useTemplateRef<SnackBarType>('snackbar');

function changeMonth(offset: number): void {
    changeBudgetMonth(offset).catch(error => {
        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function save(): void {
    saveBudgets().then(() => {
        snackbar.value?.showMessage('You have saved the budgets');
    }).catch(error => {
        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

loadAll().catch(error => {
    if (!error.processed) {
        snackbar.value?.showError(error);
    }
});
</script>

<style>
.budget-amount-column {
    min-width: 180px;
}

.budget-progress-column {
    width: 20%;
    min-width: 120px;
}
</style>
