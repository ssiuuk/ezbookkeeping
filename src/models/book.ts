export interface BookInfoResponse {
    readonly id: string;
    readonly ownerUsername: string;
    readonly ownerNickname: string;
    readonly isOwner: boolean;
}

export interface BookMemberInfoResponse {
    readonly id: string;
    readonly username: string;
    readonly nickname: string;
    readonly isOwner: boolean;
    readonly createdAt: number;
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
