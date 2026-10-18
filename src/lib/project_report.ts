import { CategoryType } from '@/core/category.ts';

import type { TransactionStatisticResponseItem } from '@/models/transaction.ts';

// converts an amount in the smallest unit of the given currency to the report currency, null when no rate is known
export type ProjectAmountConverter = (amount: number, currency: string) => number | null;

export interface ProjectReportAccount {
    readonly id: string;
    readonly currency: string;
}

export interface ProjectReportCategory {
    readonly id: string;
    readonly name: string;
    readonly type: number;
    readonly parentId: string;
    readonly icon: string;
    readonly iconType: number;
    readonly color: string;
}

export interface ProjectCategorySummary {
    readonly categoryId: string;
    readonly name: string;
    readonly icon: string;
    readonly iconType: number;
    readonly color: string;
    readonly amount: number;
    readonly percent: number;
}

export interface ProjectReport {
    readonly totalIncome: number;
    readonly totalExpense: number;
    readonly net: number;
    readonly expenseByCategory: ProjectCategorySummary[];
    readonly incomeByCategory: ProjectCategorySummary[];
    readonly incomplete: boolean;
}

function getPrimaryCategory(category: ProjectReportCategory, categoriesMap: Record<string, ProjectReportCategory>): ProjectReportCategory {
    if (category.parentId && category.parentId !== '0') {
        return categoriesMap[category.parentId] ?? category;
    }

    return category;
}

function toSortedSummaries(amounts: Record<string, number>, categoriesMap: Record<string, ProjectReportCategory>, total: number): ProjectCategorySummary[] {
    const result: ProjectCategorySummary[] = [];

    for (const [categoryId, amount] of Object.entries(amounts)) {
        const category = categoriesMap[categoryId];

        result.push({
            categoryId: categoryId,
            name: category?.name ?? '',
            icon: category?.icon ?? '',
            iconType: category?.iconType ?? 0,
            color: category?.color ?? '',
            amount: amount,
            percent: total > 0 ? amount * 100 / total : 0
        });
    }

    result.sort((a, b) => {
        if (a.amount !== b.amount) {
            return b.amount - a.amount;
        }

        return a.name.localeCompare(b.name);
    });

    return result;
}

/**
 * Adds up the income and expense of one project from transaction statistic items (which are
 * already filtered by the project), grouped by primary category. Transfers are not counted.
 * Amounts are in the smallest unit of the report currency.
 */
export function buildProjectReport(statisticItems: TransactionStatisticResponseItem[], accountsMap: Record<string, ProjectReportAccount>, categoriesMap: Record<string, ProjectReportCategory>, convert: ProjectAmountConverter): ProjectReport {
    const expenseAmounts: Record<string, number> = {};
    const incomeAmounts: Record<string, number> = {};
    let totalIncome = 0;
    let totalExpense = 0;
    let incomplete = false;

    for (const item of statisticItems) {
        const account = accountsMap[item.accountId];
        const category = categoriesMap[item.categoryId];

        if (!account || !category || (category.type !== CategoryType.Income && category.type !== CategoryType.Expense)) {
            continue;
        }

        const amount = convert(parseInt(item.amount) || 0, account.currency);

        if (amount === null) {
            incomplete = true;
            continue;
        }

        const primaryCategory = getPrimaryCategory(category, categoriesMap);

        if (category.type === CategoryType.Expense) {
            totalExpense += amount;
            expenseAmounts[primaryCategory.id] = (expenseAmounts[primaryCategory.id] ?? 0) + amount;
        } else {
            totalIncome += amount;
            incomeAmounts[primaryCategory.id] = (incomeAmounts[primaryCategory.id] ?? 0) + amount;
        }
    }

    return {
        totalIncome: totalIncome,
        totalExpense: totalExpense,
        net: totalIncome - totalExpense,
        expenseByCategory: toSortedSummaries(expenseAmounts, categoriesMap, totalExpense),
        incomeByCategory: toSortedSummaries(incomeAmounts, categoriesMap, totalIncome),
        incomplete: incomplete
    };
}
