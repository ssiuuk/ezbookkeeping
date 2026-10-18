import type { BigDecimal } from '@/core/numeral.ts';

import { BIG_DECIMAL_ZERO } from '@/lib/numeral.ts';

export interface MonthlyBudgetProgress {
    readonly usedRate: number;
    readonly remainingAmount: BigDecimal;
    readonly overBudget: boolean;
    readonly aheadOfSchedule: boolean;
    readonly dailyAllowance: BigDecimal;
    readonly projectedOverBudget: boolean;
}

export function calculateMonthlyBudgetProgress(budgetAmount: BigDecimal, expenseAmount: BigDecimal, estimatedExpense: BigDecimal, currentDay: number, daysInMonth: number): MonthlyBudgetProgress | undefined {
    if (!budgetAmount.isPositive() || daysInMonth < 1) {
        return undefined;
    }

    const usedRate = expenseAmount.divide(budgetAmount).toDoubleNumber();
    const elapsedRate = currentDay / daysInMonth;
    const remainingAmount = budgetAmount.subtract(expenseAmount);
    const remainingDays = Math.max(daysInMonth - currentDay + 1, 1);

    return {
        usedRate: usedRate,
        remainingAmount: remainingAmount,
        overBudget: remainingAmount.isNegative(),
        aheadOfSchedule: usedRate > elapsedRate,
        dailyAllowance: remainingAmount.isPositive() ? remainingAmount.divide(remainingDays) : BIG_DECIMAL_ZERO,
        projectedOverBudget: estimatedExpense.greaterThan(budgetAmount)
    };
}
