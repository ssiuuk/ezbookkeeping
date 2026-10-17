import { describe, expect, test } from 'vitest';

import type { AccountInfoResponse } from '@/models/account.ts';
import type { TransactionCategoryInfoResponse } from '@/models/transaction_category.ts';

import { buildHouseholdReport, type HouseholdAmountConverter } from '@/lib/household_report.ts';

function account(id: string, currency: string, balance: string, subAccounts?: AccountInfoResponse[]): AccountInfoResponse {
    return { id, name: id, parentId: '0', category: 1, type: 1, icon: '1', iconType: 0, color: '000000', currency, balance, comment: '', displayOrder: 1, hidden: false, subAccounts };
}

function category(id: string, name: string, type: number, parentId: string = '0', subCategories?: TransactionCategoryInfoResponse[]): TransactionCategoryInfoResponse {
    return { id, name, parentId, type, icon: '1', iconType: 0, color: '000000', comment: '', displayOrder: 1, hidden: false, subCategories };
}

// USD is the report currency, 1 EUR = 2 USD, no rate for JPY
const convert: HouseholdAmountConverter = (amount, currency) => {
    if (currency === 'USD') {
        return amount;
    } else if (currency === 'EUR') {
        return amount * 2;
    }

    return null;
};

describe('buildHouseholdReport', () => {
    const familyBook = {
        bookId: '1',
        bookName: 'Family',
        accounts: [account('a1', 'USD', '100000')],
        categories: {
            1: [category('i1', 'Salary', 1, '0', [category('i11', 'Base', 1, 'i1')])],
            2: [category('e1', 'Food', 2, '0', [category('e11', 'Groceries', 2, 'e1')]), category('e2', 'Housing', 2, '0', [category('e21', 'Rent', 2, 'e2')])]
        },
        statisticItems: [
            { categoryId: 'i11', accountId: 'a1', amount: '300000' },
            { categoryId: 'e11', accountId: 'a1', amount: '40000' },
            { categoryId: 'e21', accountId: 'a1', amount: '120000' }
        ]
    };

    const privateBook = {
        bookId: '2',
        bookName: 'My Book',
        accounts: [account('b1', 'USD', '0', [account('b11', 'EUR', '5000'), account('b12', 'USD', '-2000')])],
        categories: {
            2: [category('f1', 'Food ', 2, '0', [category('f11', 'Dining', 2, 'f1')])],
            3: [category('t1', 'Transfer', 3, '0', [category('t11', 'Move', 3, 't1')])]
        },
        statisticItems: [
            { categoryId: 'f11', accountId: 'b11', amount: '1000' },
            { categoryId: 't11', accountId: 'b12', amount: '9999' }
        ]
    };

    test('adds up income, expense, cash flow and net worth across books', () => {
        const report = buildHouseholdReport([familyBook, privateBook], convert);

        expect(report.books[0]).toEqual({ bookId: '1', bookName: 'Family', income: 300000, expense: 160000, net: 140000, netWorth: 100000, incomplete: false });
        expect(report.books[1]).toEqual({ bookId: '2', bookName: 'My Book', income: 0, expense: 2000, net: -2000, netWorth: 8000, incomplete: false });
        expect(report.totalIncome).toBe(300000);
        expect(report.totalExpense).toBe(162000);
        expect(report.totalNet).toBe(138000);
        expect(report.totalNetWorth).toBe(108000);
        expect(report.savingsRate).toBeCloseTo(46, 5);
        expect(report.incomplete).toBe(false);
    });

    test('merges expense of primary categories with the same name', () => {
        const report = buildHouseholdReport([familyBook, privateBook], convert);

        expect(report.expenseByCategory.map(item => [item.name, item.amount])).toEqual([['Housing', 120000], ['Food', 42000]]);
        expect(report.expenseByCategory[1]?.percent).toBeCloseTo(42000 / 162000 * 100, 5);
    });

    test('flags amounts in currencies without an exchange rate', () => {
        const report = buildHouseholdReport([{ ...privateBook, accounts: [account('b11', 'JPY', '100')], statisticItems: [{ categoryId: 'f11', accountId: 'b11', amount: '500' }] }], convert);

        expect(report.incomplete).toBe(true);
        expect(report.totalExpense).toBe(0);
        expect(report.savingsRate).toBeNull();
    });
});
