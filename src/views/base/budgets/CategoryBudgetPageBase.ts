import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useUserStore } from '@/stores/user.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';

import type { Year0BasedMonth } from '@/core/datetime.ts';
import type { BigDecimal } from '@/core/numeral.ts';
import { CategoryType } from '@/core/category.ts';
import { KeywordMatchMode } from '@/core/text.ts';

import type { TransactionCategory } from '@/models/transaction_category.ts';
import type { TransactionCategoryBudgetInfoResponse } from '@/models/transaction_category_budget.ts';
import type { TransactionStatisticResponse } from '@/models/transaction.ts';

import {
    type CategoryBudgetAmount,
    type CategoryBudgetSummary,
    type ConvertedCategoryBudgets,
    buildCategoryBudgetSummary,
    convertCategoryBudgetsToCurrency
} from '@/lib/category_budget.ts';
import {
    getCurrentUnixTime,
    getYear0BasedMonthObjectFromUnixTime,
    getYearMonthFirstUnixTime,
    getYearMonthLastUnixTime,
    parseDateTimeFromUnixTime
} from '@/lib/datetime.ts';
import { parseBigDecimal } from '@/lib/numeral.ts';
import { handleApiResponse } from '@/lib/api.ts';
import services from '@/lib/services.ts';

export function useCategoryBudgetPageBase() {
    const { formatDateTimeToGregorianLikeLongYearMonth, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();

    const userStore = useUserStore();
    const accountsStore = useAccountsStore();
    const transactionCategoriesStore = useTransactionCategoriesStore();
    const exchangeRatesStore = useExchangeRatesStore();

    const loading = ref<boolean>(true);
    const saving = ref<boolean>(false);
    const editing = ref<boolean>(false);
    const currentMonth = ref<Year0BasedMonth>(getYear0BasedMonthObjectFromUnixTime(getCurrentUnixTime()));
    const budgets = ref<Record<string, CategoryBudgetAmount>>({});
    const editingBudgets = ref<Record<string, number>>({});
    const statistics = ref<TransactionStatisticResponse | null>(null);

    const defaultCurrency = computed<string>(() => userStore.currentUserDefaultCurrency);
    const currentMonthDisplayName = computed<string>(() => formatDateTimeToGregorianLikeLongYearMonth(parseDateTimeFromUnixTime(getYearMonthFirstUnixTime(currentMonth.value))));

    const primaryExpenseCategories = computed<TransactionCategory[]>(() => {
        const categories = transactionCategoriesStore.allTransactionCategories[CategoryType.Expense] ?? [];
        return categories.filter(category => category.visible || (budgets.value[category.id]?.amount ?? 0) > 0);
    });

    // converts amounts the same way for spending and budgets, so that both are compared in the same currency
    function exchangeAmount(amount: BigDecimal, fromCurrency: string, toCurrency: string): BigDecimal | null {
        if (fromCurrency === toCurrency) {
            return amount;
        }

        const exchangedAmount = exchangeRatesStore.getExchangedAmount(amount, fromCurrency, toCurrency);
        return exchangedAmount ? exchangedAmount.truncate() : null;
    }

    // budgets are saved in the currency they were entered in, which may differ from the current user's default currency in a shared book
    const convertedBudgets = computed<ConvertedCategoryBudgets>(() => convertCategoryBudgetsToCurrency(budgets.value, defaultCurrency.value, (amount, fromCurrency, toCurrency) => {
        const exchangedAmount = exchangeAmount(parseBigDecimal(amount), fromCurrency, toCurrency);
        return exchangedAmount ? exchangedAmount.toDoubleNumber() : null;
    }));

    const spending = computed<{ spentByCategoryId: Record<string, number>, incomplete: boolean }>(() => {
        const spentByCategoryId: Record<string, number> = {};
        let incomplete = false;

        for (const item of statistics.value?.items ?? []) {
            const account = accountsStore.allAccountsMap[item.accountId];
            let category = transactionCategoriesStore.allTransactionCategoriesMap[item.categoryId];

            if (!account || !category || category.type !== CategoryType.Expense) {
                continue;
            }

            if (category.parentId && category.parentId !== '0') {
                category = transactionCategoriesStore.allTransactionCategoriesMap[category.parentId] ?? category;
            }

            const amount = exchangeAmount(parseBigDecimal(item.amount), account.currency, defaultCurrency.value);

            if (!amount) {
                incomplete = true;
                continue;
            }

            spentByCategoryId[category.id] = (spentByCategoryId[category.id] ?? 0) + amount.toDoubleNumber();
        }

        return {
            spentByCategoryId: spentByCategoryId,
            incomplete: incomplete
        };
    });

    const summary = computed<CategoryBudgetSummary>(() => buildCategoryBudgetSummary(primaryExpenseCategories.value, convertedBudgets.value.budgets, spending.value.spentByCategoryId));

    function setBudgets(items: TransactionCategoryBudgetInfoResponse[]): void {
        const result: Record<string, CategoryBudgetAmount> = {};

        for (const item of items) {
            result[item.categoryId] = {
                amount: item.amount,
                currency: item.currency || defaultCurrency.value
            };
        }

        budgets.value = result;
    }

    // the currency the budget of the given category is saved in (and edited in), new budgets are in the current user's default currency
    function getBudgetCurrency(categoryId: string): string {
        return budgets.value[categoryId]?.currency || defaultCurrency.value;
    }

    // the budget amount in the currency it was saved in, when that currency is not the displayed currency, otherwise empty
    function getOriginalBudgetText(categoryId: string): string {
        const budget = budgets.value[categoryId];

        if (!budget || budget.amount <= 0 || !budget.currency || budget.currency === defaultCurrency.value) {
            return '';
        }

        return formatAmount(budget.amount, budget.currency);
    }

    function loadStatistics(): Promise<void> {
        return handleApiResponse(services.getTransactionStatistics({
            startTime: getYearMonthFirstUnixTime(currentMonth.value),
            endTime: getYearMonthLastUnixTime(currentMonth.value),
            tagFilter: '',
            keyword: '',
            matchMode: KeywordMatchMode.Default.type,
            useTransactionTimezone: false
        }), 'failed to retrieve transaction statistics', 'Unable to retrieve transaction statistics').then(result => {
            statistics.value = result;
        });
    }

    function loadAll(): Promise<void> {
        loading.value = true;

        return Promise.all([
            accountsStore.loadAllAccounts({ force: false }),
            transactionCategoriesStore.loadAllCategories({ force: false }),
            handleApiResponse(services.getTransactionCategoryBudgets(), 'failed to retrieve category budgets', 'Unable to retrieve budgets'),
            loadStatistics()
        ]).then(([, , budgetItems]) => {
            setBudgets(budgetItems);
            loading.value = false;
        }).catch(error => {
            loading.value = false;
            throw error;
        });
    }

    function changeMonth(offset: number): Promise<void> {
        let month0base = currentMonth.value.month0base + offset;
        let year = currentMonth.value.year;

        while (month0base < 0) {
            month0base += 12;
            year--;
        }

        while (month0base > 11) {
            month0base -= 12;
            year++;
        }

        currentMonth.value = { year: year, month0base: month0base };
        loading.value = true;

        return loadStatistics().finally(() => {
            loading.value = false;
        });
    }

    function startEditing(): void {
        const result: Record<string, number> = {};

        for (const [categoryId, budget] of Object.entries(budgets.value)) {
            result[categoryId] = budget.amount;
        }

        editingBudgets.value = result;
        editing.value = true;
    }

    function cancelEditing(): void {
        editing.value = false;
        editingBudgets.value = {};
    }

    function saveBudgets(): Promise<void> {
        const items = Object.entries(editingBudgets.value)
            .filter(([, amount]) => amount > 0)
            .map(([categoryId, amount]) => ({ categoryId: categoryId, amount: amount, currency: getBudgetCurrency(categoryId) }));

        saving.value = true;

        return handleApiResponse(services.saveTransactionCategoryBudgets({ budgets: items }), 'failed to save category budgets', 'Unable to save budgets').then(result => {
            setBudgets(result);
            saving.value = false;
            editing.value = false;
            editingBudgets.value = {};
        }).catch(error => {
            saving.value = false;
            throw error;
        });
    }

    function formatAmount(amount: number, currency?: string): string {
        return formatAmountToLocalizedNumeralsWithCurrency(parseBigDecimal(amount), currency || defaultCurrency.value);
    }

    return {
        // states
        loading,
        saving,
        editing,
        editingBudgets,
        // computed states
        defaultCurrency,
        currentMonthDisplayName,
        summary,
        incompleteSpending: computed<boolean>(() => spending.value.incomplete),
        incompleteBudget: computed<boolean>(() => convertedBudgets.value.unconvertedCategoryIds.length > 0),
        // functions
        loadAll,
        changeMonth,
        startEditing,
        cancelEditing,
        saveBudgets,
        getBudgetCurrency,
        getOriginalBudgetText,
        formatAmount
    };
}
