import type { AccountInfoResponse } from '@/models/account.ts';
import type { TransactionCategoryInfoResponse } from '@/models/transaction_category.ts';
import type { TransactionStatisticResponseItem } from '@/models/transaction.ts';

import { CategoryType } from '@/core/category.ts';

// converts an amount in the smallest unit of the given currency to the report currency, null when no rate is known
export type HouseholdAmountConverter = (amount: number, currency: string) => number | null;

export interface HouseholdBookData {
    readonly bookId: string;
    readonly bookName: string;
    readonly accounts: AccountInfoResponse[];
    readonly categories: Record<number, TransactionCategoryInfoResponse[]>;
    readonly statisticItems: TransactionStatisticResponseItem[];
}

export interface HouseholdBookSummary {
    readonly bookId: string;
    readonly bookName: string;
    readonly income: number;
    readonly expense: number;
    readonly net: number;
    readonly netWorth: number;
    readonly incomplete: boolean;
}

export interface HouseholdCategorySummary {
    readonly name: string;
    readonly amount: number;
    readonly percent: number;
}

export interface HouseholdReport {
    readonly books: HouseholdBookSummary[];
    readonly totalIncome: number;
    readonly totalExpense: number;
    readonly totalNet: number;
    readonly totalNetWorth: number;
    readonly savingsRate: number | null;
    readonly expenseByCategory: HouseholdCategorySummary[];
    readonly incomplete: boolean;
}

function flattenAccounts(accounts: AccountInfoResponse[]): AccountInfoResponse[] {
    const result: AccountInfoResponse[] = [];

    for (const account of accounts) {
        if (account.subAccounts && account.subAccounts.length) {
            result.push(...account.subAccounts);
        } else {
            result.push(account);
        }
    }

    return result;
}

function getCategoryMap(categories: Record<number, TransactionCategoryInfoResponse[]>): Record<string, TransactionCategoryInfoResponse> {
    const result: Record<string, TransactionCategoryInfoResponse> = {};

    for (const primaryCategories of Object.values(categories)) {
        for (const category of primaryCategories) {
            result[category.id] = category;

            for (const subCategory of category.subCategories ?? []) {
                result[subCategory.id] = subCategory;
            }
        }
    }

    return result;
}

/**
 * Adds up income, expense, net cash flow and net worth of every book, and merges expense of
 * primary categories with the same name across books. Amounts are in the smallest unit of the
 * report currency.
 */
export function buildHouseholdReport(books: HouseholdBookData[], convert: HouseholdAmountConverter): HouseholdReport {
    const bookSummaries: HouseholdBookSummary[] = [];
    const expenseByCategoryName: Record<string, number> = {};
    let totalIncome = 0;
    let totalExpense = 0;
    let totalNetWorth = 0;
    let incomplete = false;

    for (const book of books) {
        const leafAccounts = flattenAccounts(book.accounts);
        const accountMap: Record<string, AccountInfoResponse> = {};
        const categoryMap = getCategoryMap(book.categories);
        let income = 0;
        let expense = 0;
        let netWorth = 0;
        let bookIncomplete = false;

        for (const account of leafAccounts) {
            accountMap[account.id] = account;

            const balance = convert(parseInt(account.balance) || 0, account.currency);

            if (balance === null) {
                bookIncomplete = true;
                continue;
            }

            netWorth += balance;
        }

        for (const item of book.statisticItems) {
            const account = accountMap[item.accountId];
            const category = categoryMap[item.categoryId];

            if (!account || !category || (category.type !== CategoryType.Income && category.type !== CategoryType.Expense)) {
                continue;
            }

            const amount = convert(parseInt(item.amount) || 0, account.currency);

            if (amount === null) {
                bookIncomplete = true;
                continue;
            }

            if (category.type === CategoryType.Income) {
                income += amount;
            } else {
                expense += amount;

                const primaryCategory = category.parentId && category.parentId !== '0' ? (categoryMap[category.parentId] ?? category) : category;
                const name = primaryCategory.name.trim();
                expenseByCategoryName[name] = (expenseByCategoryName[name] ?? 0) + amount;
            }
        }

        bookSummaries.push({
            bookId: book.bookId,
            bookName: book.bookName,
            income: income,
            expense: expense,
            net: income - expense,
            netWorth: netWorth,
            incomplete: bookIncomplete
        });

        totalIncome += income;
        totalExpense += expense;
        totalNetWorth += netWorth;
        incomplete = incomplete || bookIncomplete;
    }

    const expenseByCategory: HouseholdCategorySummary[] = Object.entries(expenseByCategoryName)
        .map(([name, amount]) => ({ name: name, amount: amount, percent: totalExpense > 0 ? amount / totalExpense * 100 : 0 }))
        .sort((a, b) => b.amount - a.amount);

    return {
        books: bookSummaries,
        totalIncome: totalIncome,
        totalExpense: totalExpense,
        totalNet: totalIncome - totalExpense,
        totalNetWorth: totalNetWorth,
        savingsRate: totalIncome > 0 ? (totalIncome - totalExpense) / totalIncome * 100 : null,
        expenseByCategory: expenseByCategory,
        incomplete: incomplete
    };
}
