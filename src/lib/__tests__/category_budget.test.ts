import { describe, expect, test } from 'vitest';

import {
    type CategoryBudgetAmountExchanger,
    type CategoryBudgetCategory,
    buildCategoryBudgetSummary,
    convertCategoryBudgetsToCurrency
} from '@/lib/category_budget.ts';
import { getExchangedAmountByRate, parseBigDecimal } from '@/lib/numeral.ts';

function category(id: string, name: string): CategoryBudgetCategory {
    return { id, name, icon: '1', iconType: 0, color: '000000' };
}

describe('buildCategoryBudgetSummary', () => {
    const food = category('1', 'Food');
    const travel = category('2', 'Travel');
    const gifts = category('3', 'Gifts');

    test('compares spending with each budget', () => {
        const summary = buildCategoryBudgetSummary([food, travel], { '1': 50000, '2': 20000 }, { '1': 12500, '2': 25000 });

        expect(summary.rows[0]).toEqual({ category: food, budget: 50000, spent: 12500, remaining: 37500, usedRate: 25, overBudget: false });
        expect(summary.rows[1]).toEqual({ category: travel, budget: 20000, spent: 25000, remaining: -5000, usedRate: 125, overBudget: true });
        expect(summary.totalBudget).toBe(70000);
        expect(summary.totalSpent).toBe(37500);
        expect(summary.totalRemaining).toBe(32500);
        expect(summary.totalUsedRate).toBeCloseTo(53.5714, 3);
        expect(summary.totalOverBudget).toBe(false);
    });

    test('keeps spending of categories without a budget out of the totals', () => {
        const summary = buildCategoryBudgetSummary([food, gifts], { '1': 10000 }, { '1': 4000, '3': 3000 });

        expect(summary.rows[1]).toEqual({ category: gifts, budget: 0, spent: 3000, remaining: -3000, usedRate: 0, overBudget: false });
        expect(summary.totalBudget).toBe(10000);
        expect(summary.totalSpent).toBe(4000);
        expect(summary.unbudgetedSpent).toBe(3000);
    });

    test('handles no budgets and no spending', () => {
        const summary = buildCategoryBudgetSummary([food], {}, {});

        expect(summary.rows[0]?.usedRate).toBe(0);
        expect(summary.totalUsedRate).toBe(0);
        expect(summary.totalOverBudget).toBe(false);
        expect(summary.unbudgetedSpent).toBe(0);
    });

    test('marks the total over budget', () => {
        const summary = buildCategoryBudgetSummary([food, travel], { '1': 1000, '2': 1000 }, { '1': 1500, '2': 900 });

        expect(summary.totalOverBudget).toBe(true);
        expect(summary.totalRemaining).toBe(-400);
    });
});

// exchange rates against USD, 1 USD = 0.8 GBP, converted the same way as the exchange rates store does
const exchangeRates: Record<string, string> = { USD: '1', GBP: '0.8' };

const exchanger: CategoryBudgetAmountExchanger = (amount, fromCurrency, toCurrency) => {
    const fromRate = exchangeRates[fromCurrency];
    const toRate = exchangeRates[toCurrency];

    if (!fromRate || !toRate) {
        return null;
    }

    const exchangedAmount = getExchangedAmountByRate(parseBigDecimal(amount), fromRate, toRate);
    return exchangedAmount ? exchangedAmount.truncate().toDoubleNumber() : null;
};

describe('convertCategoryBudgetsToCurrency', () => {
    const food = category('1', 'Food');

    test('keeps budgets which are in the target currency', () => {
        const result = convertCategoryBudgetsToCurrency({ '1': { amount: 10000, currency: 'GBP' }, '2': { amount: 500, currency: '' } }, 'GBP', exchanger);

        expect(result.budgets).toEqual({ '1': 10000, '2': 500 });
        expect(result.unconvertedCategoryIds).toEqual([]);
    });

    test('converts budgets from the currency they were saved in', () => {
        const result = convertCategoryBudgetsToCurrency({ '1': { amount: 10000, currency: 'GBP' }, '2': { amount: 5000, currency: 'USD' } }, 'USD', exchanger);

        expect(result.budgets).toEqual({ '1': 12500, '2': 5000 });
        expect(result.unconvertedCategoryIds).toEqual([]);
    });

    test('reports budgets without exchange rate', () => {
        const result = convertCategoryBudgetsToCurrency({ '1': { amount: 10000, currency: 'JPY' }, '2': { amount: 5000, currency: 'GBP' } }, 'USD', exchanger);

        expect(result.budgets).toEqual({ '2': 6250 });
        expect(result.unconvertedCategoryIds).toEqual(['1']);
    });

    test('members with different default currencies get the same utilization of a shared GBP 100 budget', () => {
        // a member with GBP as default currency saved a GBP 100.00 budget, GBP 80.00 has been spent from a GBP account
        const savedBudgets = { '1': { amount: 10000, currency: 'GBP' } };
        const spentInGbp = 8000;

        // the GBP member sees everything in GBP
        const gbpBudgets = convertCategoryBudgetsToCurrency(savedBudgets, 'GBP', exchanger);
        const gbpSummary = buildCategoryBudgetSummary([food], gbpBudgets.budgets, { '1': spentInGbp });

        // the USD member sees the spending and the budget both converted into USD
        const spentInUsd = exchanger(spentInGbp, 'GBP', 'USD') as number;
        const usdBudgets = convertCategoryBudgetsToCurrency(savedBudgets, 'USD', exchanger);
        const usdSummary = buildCategoryBudgetSummary([food], usdBudgets.budgets, { '1': spentInUsd });

        expect(gbpSummary.rows[0]).toEqual({ category: food, budget: 10000, spent: 8000, remaining: 2000, usedRate: 80, overBudget: false });
        expect(usdSummary.rows[0]).toEqual({ category: food, budget: 12500, spent: 10000, remaining: 2500, usedRate: 80, overBudget: false });
        expect(usdSummary.totalUsedRate).toBe(gbpSummary.totalUsedRate);
        expect(usdSummary.totalOverBudget).toBe(gbpSummary.totalOverBudget);

        // reading the stored amount unchanged as USD 100.00 would show the budget as fully used
        const unconvertedSummary = buildCategoryBudgetSummary([food], { '1': 10000 }, { '1': spentInUsd });
        expect(unconvertedSummary.rows[0]?.usedRate).toBe(100);
    });

    test('changing the default currency does not reinterpret existing budgets', () => {
        const savedBudgets = { '1': { amount: 10000, currency: 'GBP' } };

        expect(convertCategoryBudgetsToCurrency(savedBudgets, 'GBP', exchanger).budgets['1']).toBe(10000);
        expect(convertCategoryBudgetsToCurrency(savedBudgets, 'USD', exchanger).budgets['1']).toBe(12500);
    });
});
