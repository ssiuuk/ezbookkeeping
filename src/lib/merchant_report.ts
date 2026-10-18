import { TransactionType } from '@/core/transaction.ts';

import type { TransactionMerchantStatisticResponseItem } from '@/models/transaction.ts';

// converts an amount in the smallest unit of the given currency to the report currency, null when no rate is known
export type MerchantAmountConverter = (amount: number, currency: string) => number | null;

export interface MerchantReportAccount {
    readonly id: string;
    readonly currency: string;
}

export interface MerchantSummary {
    readonly merchant: string; // empty for transactions without merchant
    readonly expense: number;
    readonly income: number;
    readonly count: number;
    readonly expensePercent: number;
}

export interface MerchantReport {
    readonly totalExpense: number;
    readonly totalIncome: number;
    readonly totalCount: number;
    readonly items: MerchantSummary[];
    readonly incomplete: boolean;
}

interface MerchantTotal {
    expense: number;
    income: number;
    count: number;
}

/**
 * Adds up the income and expense of every merchant from merchant statistic items (which are grouped
 * by merchant, transaction type and account). Merchants are sorted by expense, then by income, and
 * transactions without merchant come last. Amounts are in the smallest unit of the report currency.
 */
export function buildMerchantReport(statisticItems: TransactionMerchantStatisticResponseItem[], accountsMap: Record<string, MerchantReportAccount>, convert: MerchantAmountConverter): MerchantReport {
    const totals: Record<string, MerchantTotal> = {};
    let totalExpense = 0;
    let totalIncome = 0;
    let totalCount = 0;
    let incomplete = false;

    for (const item of statisticItems) {
        const account = accountsMap[item.accountId];

        if (!account || (item.type !== TransactionType.Income && item.type !== TransactionType.Expense)) {
            continue;
        }

        const merchant = (item.merchant ?? '').trim();
        const total = totals[merchant] ?? (totals[merchant] = { expense: 0, income: 0, count: 0 });
        const count = item.count || 0;

        total.count += count;
        totalCount += count;

        const amount = convert(parseInt(item.amount) || 0, account.currency);

        if (amount === null) {
            incomplete = true;
            continue;
        }

        if (item.type === TransactionType.Expense) {
            total.expense += amount;
            totalExpense += amount;
        } else {
            total.income += amount;
            totalIncome += amount;
        }
    }

    const items: MerchantSummary[] = [];

    for (const [merchant, total] of Object.entries(totals)) {
        items.push({
            merchant: merchant,
            expense: total.expense,
            income: total.income,
            count: total.count,
            expensePercent: totalExpense > 0 ? total.expense * 100 / totalExpense : 0
        });
    }

    items.sort((a, b) => {
        if (!a.merchant !== !b.merchant) {
            return a.merchant ? -1 : 1;
        }

        if (a.expense !== b.expense) {
            return b.expense - a.expense;
        }

        if (a.income !== b.income) {
            return b.income - a.income;
        }

        return a.merchant.localeCompare(b.merchant);
    });

    return {
        totalExpense: totalExpense,
        totalIncome: totalIncome,
        totalCount: totalCount,
        items: items,
        incomplete: incomplete
    };
}
