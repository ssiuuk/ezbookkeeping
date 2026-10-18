import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { usePeriodStatisticsWidgetBase } from '@/views/base/overview/PeriodStatisticsWidgetBase.ts';

import type { BigDecimal } from '@/core/numeral.ts';
import { type DateTime, DateRange } from '@/core/datetime.ts';

import { DISPLAY_HIDDEN_PERCENT } from '@/consts/numeral.ts';

import { isInteger } from '@/lib/common.ts';
import { BIG_DECIMAL_ZERO, parseBigDecimal } from '@/lib/numeral.ts';
import { getCurrentDateTime } from '@/lib/datetime.ts';
import { type MonthlyBudgetProgress, calculateMonthlyBudgetProgress } from '@/lib/budget.ts';

export interface CommonMonthlyExpenseProgressWidgetProps {
    budgetAmount?: number;
}

export function useMonthlyExpenseProgressWidgetBase(props: CommonMonthlyExpenseProgressWidgetProps) {
    const { formatPercentToLocalizedNumerals } = useI18n();
    const {
        showAmountInHomePage,
        transactionOverview,
        currentExpenseAmount,
        currentDisplayExpenseAmount,
        getDisplayAmount
    } = usePeriodStatisticsWidgetBase({
        dateType: DateRange.ThisMonth.type
    });

    const currentDateTime = ref<DateTime>(getCurrentDateTime());

    const currentMonthElapsedPercent = computed<number>(() => currentDateTime.value.getGregorianCalendarDay() / currentDateTime.value.getMaxDayOfGregorianCalendarMonth());
    const estimatedExpense = computed<BigDecimal>(() => currentExpenseAmount.value.divide(currentMonthElapsedPercent.value));
    const lastMonthExpense = computed<BigDecimal>(() => transactionOverview.value.lastMonth?.expenseAmount ?? BIG_DECIMAL_ZERO);
    const incompleteThisMonthExpense = computed<boolean>(() => !!transactionOverview.value.thisMonth?.incompleteExpenseAmount);

    const displayElapsedPercent = computed<string>(() => formatPercentToLocalizedNumerals(currentMonthElapsedPercent.value * 100.0, 0, '<1'));
    const displayEstimatedExpense = computed<string>(() => getDisplayAmount(estimatedExpense.value, incompleteThisMonthExpense.value));
    const displayLastMonthExpense = computed<string>(() => getDisplayAmount(lastMonthExpense.value, !!transactionOverview.value.lastMonth?.incompleteExpenseAmount));

    const budgetAmount = computed<BigDecimal>(() => isInteger(props.budgetAmount) ? parseBigDecimal(props.budgetAmount) : BIG_DECIMAL_ZERO);
    const budgetProgress = computed<MonthlyBudgetProgress | undefined>(() => calculateMonthlyBudgetProgress(
        budgetAmount.value,
        currentExpenseAmount.value,
        estimatedExpense.value,
        currentDateTime.value.getGregorianCalendarDay(),
        currentDateTime.value.getMaxDayOfGregorianCalendarMonth()
    ));

    const displayBudgetAmount = computed<string>(() => getDisplayAmount(budgetAmount.value, false));
    const displayBudgetUsedPercent = computed<string>(() => {
        if (!showAmountInHomePage.value) {
            return DISPLAY_HIDDEN_PERCENT;
        }

        return formatPercentToLocalizedNumerals((budgetProgress.value?.usedRate ?? 0) * 100.0, 0, '<1');
    });
    const displayBudgetRemainingAmount = computed<string>(() => getDisplayAmount(budgetProgress.value?.remainingAmount.abs() ?? BIG_DECIMAL_ZERO, incompleteThisMonthExpense.value));
    const displayDailyAllowance = computed<string>(() => getDisplayAmount(budgetProgress.value?.dailyAllowance ?? BIG_DECIMAL_ZERO, incompleteThisMonthExpense.value));

    return {
        // computed states
        transactionOverview,
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
    };
}
