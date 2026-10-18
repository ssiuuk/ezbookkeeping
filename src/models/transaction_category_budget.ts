export interface TransactionCategoryBudgetItemRequest {
    readonly categoryId: string;
    readonly amount: number;
    readonly currency?: string;
}

export interface TransactionCategoryBudgetSaveRequest {
    readonly budgets: TransactionCategoryBudgetItemRequest[];
}

export interface TransactionCategoryBudgetInfoResponse {
    readonly categoryId: string;
    readonly amount: number;
    readonly currency: string;
}
