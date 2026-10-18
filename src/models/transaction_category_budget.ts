export interface TransactionCategoryBudgetItemRequest {
    readonly categoryId: string;
    readonly amount: number;
}

export interface TransactionCategoryBudgetSaveRequest {
    readonly budgets: TransactionCategoryBudgetItemRequest[];
}

export interface TransactionCategoryBudgetInfoResponse {
    readonly categoryId: string;
    readonly amount: number;
}
