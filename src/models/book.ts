export interface BookInfoResponse {
    readonly id: string;
    readonly name: string;
    readonly ownerUsername: string;
    readonly ownerNickname: string;
    readonly isPersonal: boolean;
    readonly isOwner: boolean;
}

export interface BookMemberInfoResponse {
    readonly id: string;
    readonly username: string;
    readonly nickname: string;
    readonly isOwner: boolean;
    readonly createdAt: number;
}

export interface BookSharedAccountInfoResponse {
    readonly accountId: string;
    readonly ownerId: string;
    readonly ownerUsername: string;
    readonly ownerNickname: string;
}

export interface BookCreateRequest {
    readonly name: string;
}

export interface BookModifyRequest {
    readonly id: string;
    readonly name: string;
}

export interface BookDeleteRequest {
    readonly id: string;
}

export interface BookMemberAddRequest {
    readonly username: string;
}

export interface BookMemberRemoveRequest {
    readonly memberId: string;
}

export interface BookLeaveRequest {
    readonly bookId: string;
}

export interface BookAccountShareRequest {
    readonly accountId: string;
}

export interface BookSettleUpRequest {
    readonly startTime?: number;
    readonly endTime?: number;
}

export interface BookSettleUpMemberResponse {
    readonly id: string;
    readonly username: string;
    readonly nickname: string;
    readonly isOwner: boolean;
    readonly isFormer: boolean;
}

export interface BookSettleUpMemberBalance {
    readonly memberId: string;
    readonly paid: number;
    readonly fairShare: number;
    readonly settlementsSent: number;
    readonly settlementsReceived: number;
    readonly balance: number;
}

export interface BookSettleUpPayment {
    readonly fromId: string;
    readonly toId: string;
    readonly amount: number;
}

export interface BookSettleUpCurrencyResponse {
    readonly currency: string;
    readonly totalExpense: number;
    readonly balances: BookSettleUpMemberBalance[];
    readonly suggestedPayments: BookSettleUpPayment[];
}

export interface BookSettleUpResponse {
    readonly defaultCurrency: string;
    readonly startTime: number;
    readonly endTime: number;
    readonly members: BookSettleUpMemberResponse[];
    readonly currencies: BookSettleUpCurrencyResponse[];
}

export interface BookSettlementCreateRequest {
    readonly fromId: string;
    readonly toId: string;
    readonly amount: number;
    readonly currency: string;
    readonly time?: number;
    readonly comment?: string;
}

export interface BookSettlementDeleteRequest {
    readonly id: string;
}

export interface BookSettlementInfoResponse {
    readonly id: string;
    readonly fromId: string;
    readonly toId: string;
    readonly amount: number;
    readonly currency: string;
    readonly time: number;
    readonly createdById: string;
    readonly comment: string;
    readonly createdAt: number;
}
