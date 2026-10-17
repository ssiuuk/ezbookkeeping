import { describe, expect, test } from 'vitest';

import { CategoryType } from '@/core/category.ts';
import { NO_PROJECT_ID, isProjectIdSet } from '@/models/project.ts';
import {
    type ProjectReportAccount,
    type ProjectReportCategory,
    buildProjectReport
} from '@/lib/project_report.ts';

function category(id: string, name: string, type: CategoryType, parentId: string = '0'): ProjectReportCategory {
    return { id, name, type, parentId, icon: '1', iconType: 0, color: '000000' };
}

const accounts: Record<string, ProjectReportAccount> = {
    'a1': { id: 'a1', currency: 'USD' },
    'a2': { id: 'a2', currency: 'EUR' },
    'a3': { id: 'a3', currency: 'JPY' }
};

const categories: Record<string, ProjectReportCategory> = {
    'e1': category('e1', 'Materials', CategoryType.Expense),
    'e1a': category('e1a', 'Tiles', CategoryType.Expense, 'e1'),
    'e1b': category('e1b', 'Paint', CategoryType.Expense, 'e1'),
    'e2': category('e2', 'Labor', CategoryType.Expense),
    'e2a': category('e2a', 'Plumber', CategoryType.Expense, 'e2'),
    'i1': category('i1', 'Refunds', CategoryType.Income),
    'i1a': category('i1a', 'Store Refund', CategoryType.Income, 'i1'),
    't1': category('t1', 'Transfers', CategoryType.Transfer),
    't1a': category('t1a', 'Bank Transfer', CategoryType.Transfer, 't1')
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

describe('buildProjectReport', () => {
    test('adds up income and expense by primary category', () => {
        const report = buildProjectReport([
            { categoryId: 'e1a', accountId: 'a1', amount: '30000' },
            { categoryId: 'e1b', accountId: 'a1', amount: '10000' },
            { categoryId: 'e2a', accountId: 'a1', amount: '60000' },
            { categoryId: 'i1a', accountId: 'a1', amount: '5000' }
        ], accounts, categories, convert);

        expect(report.totalExpense).toBe(100000);
        expect(report.totalIncome).toBe(5000);
        expect(report.net).toBe(-95000);
        expect(report.incomplete).toBe(false);

        expect(report.expenseByCategory.map(item => [item.categoryId, item.name, item.amount, item.percent])).toEqual([
            ['e2', 'Labor', 60000, 60],
            ['e1', 'Materials', 40000, 40]
        ]);
        expect(report.incomeByCategory.map(item => [item.categoryId, item.amount, item.percent])).toEqual([
            ['i1', 5000, 100]
        ]);
    });

    test('converts other currencies and ignores transfers', () => {
        const report = buildProjectReport([
            { categoryId: 'e1a', accountId: 'a2', amount: '1000' },
            { categoryId: 'e1a', accountId: 'a1', amount: '500' },
            { categoryId: 't1a', accountId: 'a1', amount: '99999' }
        ], accounts, categories, convert);

        expect(report.totalExpense).toBe(2500);
        expect(report.totalIncome).toBe(0);
        expect(report.expenseByCategory).toHaveLength(1);
        expect(report.incomeByCategory).toHaveLength(0);
        expect(report.incomplete).toBe(false);
    });

    test('marks the report incomplete when an amount cannot be converted', () => {
        const report = buildProjectReport([
            { categoryId: 'e1a', accountId: 'a3', amount: '1000' },
            { categoryId: 'e2a', accountId: 'a1', amount: '700' }
        ], accounts, categories, convert);

        expect(report.totalExpense).toBe(700);
        expect(report.incomplete).toBe(true);
    });

    test('skips items with unknown accounts or categories', () => {
        const report = buildProjectReport([
            { categoryId: 'unknown', accountId: 'a1', amount: '1000' },
            { categoryId: 'e1a', accountId: 'unknown', amount: '1000' }
        ], accounts, categories, convert);

        expect(report.totalExpense).toBe(0);
        expect(report.totalIncome).toBe(0);
        expect(report.net).toBe(0);
        expect(report.expenseByCategory).toEqual([]);
    });

    test('returns an empty report without items', () => {
        const report = buildProjectReport([], accounts, categories, convert);

        expect(report).toEqual({
            totalIncome: 0,
            totalExpense: 0,
            net: 0,
            expenseByCategory: [],
            incomeByCategory: [],
            incomplete: false
        });
    });
});

describe('isProjectIdSet', () => {
    test('treats empty and zero ids as no project', () => {
        expect(isProjectIdSet(undefined)).toBe(false);
        expect(isProjectIdSet(null)).toBe(false);
        expect(isProjectIdSet('')).toBe(false);
        expect(isProjectIdSet(NO_PROJECT_ID)).toBe(false);
        expect(isProjectIdSet('1234')).toBe(true);
    });
});
