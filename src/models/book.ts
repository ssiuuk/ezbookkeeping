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
