import { describe, expect, test } from 'vitest';

import { buildCategoryBudgetSummary, type CategoryBudgetCategory } from '@/lib/category_budget.ts';

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
