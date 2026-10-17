export interface CategoryBudgetCategory {
    readonly id: string;
    readonly name: string;
    readonly icon: string;
    readonly iconType: number;
    readonly color: string;
}

export interface CategoryBudgetRow {
    readonly category: CategoryBudgetCategory;
    readonly budget: number;
    readonly spent: number;
    readonly remaining: number;
    readonly usedRate: number;
    readonly overBudget: boolean;
}

export interface CategoryBudgetSummary {
    readonly rows: CategoryBudgetRow[];
    readonly totalBudget: number;
    readonly totalSpent: number;
    readonly totalRemaining: number;
    readonly totalUsedRate: number;
    readonly totalOverBudget: boolean;
    readonly unbudgetedSpent: number;
}

/**
 * Builds the budget vs actual rows of the given primary expense categories.
 * Amounts are in the smallest currency unit. Spending of categories without a budget is
 * shown in its row and also reported as unbudgeted spending, but not counted in the totals.
 */
export function buildCategoryBudgetSummary(categories: CategoryBudgetCategory[], budgets: Record<string, number>, spentByCategoryId: Record<string, number>): CategoryBudgetSummary {
    const rows: CategoryBudgetRow[] = [];
    let totalBudget = 0;
    let totalSpent = 0;
    let unbudgetedSpent = 0;

    for (const category of categories) {
        const budget = Math.max(budgets[category.id] ?? 0, 0);
        const spent = spentByCategoryId[category.id] ?? 0;

        if (budget > 0) {
            totalBudget += budget;
            totalSpent += spent;
        } else {
            unbudgetedSpent += spent;
        }

        rows.push({
            category: category,
            budget: budget,
            spent: spent,
            remaining: budget - spent,
            usedRate: getUsedRate(spent, budget),
            overBudget: budget > 0 && spent > budget
        });
    }

    return {
        rows: rows,
        totalBudget: totalBudget,
        totalSpent: totalSpent,
        totalRemaining: totalBudget - totalSpent,
        totalUsedRate: getUsedRate(totalSpent, totalBudget),
        totalOverBudget: totalBudget > 0 && totalSpent > totalBudget,
        unbudgetedSpent: unbudgetedSpent
    };
}

function getUsedRate(spent: number, budget: number): number {
    if (budget <= 0) {
        return 0;
    }

    return spent / budget * 100;
}
