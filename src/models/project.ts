export const NO_PROJECT_ID = '0';

export interface ProjectCreateRequest {
    readonly name: string;
    readonly comment: string;
}

export interface ProjectModifyRequest {
    readonly id: string;
    readonly name: string;
    readonly comment: string;
}

export interface ProjectHideRequest {
    readonly id: string;
    readonly hidden: boolean;
}

export interface ProjectNewDisplayOrderRequest {
    readonly id: string;
    readonly displayOrder: number;
}

export interface ProjectMoveRequest {
    readonly newDisplayOrders: ProjectNewDisplayOrderRequest[];
}

export interface ProjectDeleteRequest {
    readonly id: string;
}

export interface ProjectInfoResponse {
    readonly id: string;
    readonly name: string;
    readonly comment: string;
    readonly displayOrder: number;
    readonly hidden: boolean;
    readonly transactionCount: number;
    readonly firstTransactionTime?: number;
    readonly lastTransactionTime?: number;
}

export function isProjectIdSet(projectId?: string | null): projectId is string {
    return !!projectId && projectId !== NO_PROJECT_ID;
}
