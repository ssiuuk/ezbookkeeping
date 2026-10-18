import { describe, expect, test } from 'vitest';

import { TransactionType } from '@/core/transaction.ts';
import type { TransactionMerchantStatisticResponseItem } from '@/models/transaction.ts';
import {
    type MerchantReportAccount,
    buildMerchantReport
} from '@/lib/merchant_report.ts';

function item(merchant: string, type: TransactionType, accountId: string, amount: number, count: number): TransactionMerchantStatisticResponseItem {
    return { merchant, type, accountId, amount: amount.toString(), count };
}

const accounts: Record<string, MerchantReportAccount> = {
    'a1': { id: 'a1', currency: 'USD' },
    'a2': { id: 'a2', currency: 'EUR' },
    'a3': { id: 'a3', currency: 'JPY' }
};

// 1 EUR = 2 USD, JPY has no known rate
function convert(amount: number, currency: string): number | null {
    if (currency === 'USD') {
        return amount;
    } else if (currency === 'EUR') {
        return amount * 2;
    }

    return null;
}

describe('buildMerchantReport', () => {
    test('adds up expense, income and count by merchant and sorts by expense', () => {
        const report = buildMerchantReport([
            item('Coffee Shop', TransactionType.Expense, 'a1', 1500, 3),
            item('Supermarket', TransactionType.Expense, 'a1', 6000, 2),
            item('Supermarket', TransactionType.Income, 'a1', 500, 1),
            item('Employer', TransactionType.Income, 'a1', 300000, 1),
            item('Coffee Shop', TransactionType.Expense, 'a2', 1000, 1)
        ], accounts, convert);

        expect(report.totalExpense).toBe(9500);
        expect(report.totalIncome).toBe(300500);
        expect(report.totalCount).toBe(8);
        expect(report.incomplete).toBe(false);

        expect(report.items.map(summary => [summary.merchant, summary.expense, summary.income, summary.count])).toEqual([
            ['Supermarket', 6000, 500, 3],
            ['Coffee Shop', 3500, 0, 4],
            ['Employer', 0, 300000, 1]
        ]);
        expect(report.items[0]?.expensePercent).toBeCloseTo(6000 * 100 / 9500);
        expect(report.items[2]?.expensePercent).toBe(0);
    });

    test('puts transactions without merchant last', () => {
        const report = buildMerchantReport([
            item('', TransactionType.Expense, 'a1', 100000, 20),
            item('Bakery', TransactionType.Expense, 'a1', 300, 1),
            item('  ', TransactionType.Expense, 'a1', 200, 1)
        ], accounts, convert);

        expect(report.items.map(summary => [summary.merchant, summary.expense, summary.count])).toEqual([
            ['Bakery', 300, 1],
            ['', 100200, 21]
        ]);
    });

    test('ignores transfers, balance modifications and unknown accounts', () => {
        const report = buildMerchantReport([
            item('Bank', TransactionType.Transfer, 'a1', 1000, 1),
            item('Bank', TransactionType.ModifyBalance, 'a1', 1000, 1),
            item('Bakery', TransactionType.Expense, 'unknown', 1000, 1)
        ], accounts, convert);

        expect(report.totalExpense).toBe(0);
        expect(report.totalCount).toBe(0);
        expect(report.items).toEqual([]);
    });

    test('marks the report incomplete when an amount cannot be converted but still counts the transactions', () => {
        const report = buildMerchantReport([
            item('Ramen Bar', TransactionType.Expense, 'a3', 1200, 2),
            item('Ramen Bar', TransactionType.Expense, 'a1', 700, 1)
        ], accounts, convert);

        expect(report.incomplete).toBe(true);
        expect(report.totalExpense).toBe(700);
        expect(report.items.map(summary => [summary.merchant, summary.expense, summary.count])).toEqual([
            ['Ramen Bar', 700, 3]
        ]);
    });

    test('returns an empty report without items', () => {
        expect(buildMerchantReport([], accounts, convert)).toEqual({
            totalExpense: 0,
            totalIncome: 0,
            totalCount: 0,
            items: [],
            incomplete: false
        });
    });
});
