export interface BudgetAlertInfoResponse {
    readonly id: string;
    readonly categoryId: string;
    readonly categoryName: string;
    readonly period: string;
    readonly threshold: number;
    readonly spent: number;
    readonly budget: number;
    readonly currency: string;
    readonly dismissed: boolean;
    readonly time: number;
}

export interface BudgetAlertDismissRequest {
    readonly ids: string[];
}
