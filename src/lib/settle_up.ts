import type {
    BookSettleUpMemberResponse,
    BookSettleUpResponse,
    BookSettlementInfoResponse
} from '@/models/book.ts';

export interface SettleUpBalanceItem {
    readonly memberId: string;
    readonly memberName: string;
    readonly isFormer: boolean;
    readonly paid: number;
    readonly fairShare: number;
    readonly settled: number; // settlements paid minus settlements received
    readonly balance: number; // positive means the member is owed, negative means the member owes
}

export interface SettleUpPaymentItem {
    readonly fromId: string;
    readonly fromName: string;
    readonly toId: string;
    readonly toName: string;
    readonly amount: number;
    readonly currency: string;
}

export interface SettleUpCurrencyItem {
    readonly currency: string;
    readonly totalExpense: number;
    readonly balances: SettleUpBalanceItem[];
    readonly payments: SettleUpPaymentItem[];
    readonly settled: boolean;
}

export interface SettlementItem {
    readonly id: string;
    readonly fromName: string;
    readonly toName: string;
    readonly amount: number;
    readonly currency: string;
    readonly time: number;
    readonly comment: string;
    readonly canDelete: boolean;
}

export function getSettleUpMemberName(member: BookSettleUpMemberResponse | undefined, fallbackName: string): string {
    if (!member) {
        return fallbackName;
    }

    return member.nickname || member.username || fallbackName;
}

export function getSettleUpMemberMap(members: BookSettleUpMemberResponse[]): Record<string, BookSettleUpMemberResponse> {
    const result: Record<string, BookSettleUpMemberResponse> = {};

    for (const member of members) {
        result[member.id] = member;
    }

    return result;
}

// the server settles up each currency on its own and puts the default currency of the book first, the order is kept
export function buildSettleUpCurrencyItems(settleUp: BookSettleUpResponse, unknownMemberName: string): SettleUpCurrencyItem[] {
    const members = getSettleUpMemberMap(settleUp.members);

    return settleUp.currencies.map(currency => ({
        currency: currency.currency,
        totalExpense: currency.totalExpense,
        balances: currency.balances.map(balance => ({
            memberId: balance.memberId,
            memberName: getSettleUpMemberName(members[balance.memberId], unknownMemberName),
            isFormer: !!members[balance.memberId]?.isFormer,
            paid: balance.paid,
            fairShare: balance.fairShare,
            settled: balance.settlementsSent - balance.settlementsReceived,
            balance: balance.balance
        })),
        payments: currency.suggestedPayments.map(payment => ({
            fromId: payment.fromId,
            fromName: getSettleUpMemberName(members[payment.fromId], unknownMemberName),
            toId: payment.toId,
            toName: getSettleUpMemberName(members[payment.toId], unknownMemberName),
            amount: payment.amount,
            currency: currency.currency
        })),
        settled: currency.suggestedPayments.length < 1
    }));
}

// matches what the server allows: the book owner, or the member who recorded, paid or received the settlement
export function canDeleteSettlement(settlement: BookSettlementInfoResponse, currentMemberId: string | undefined, isBookOwner: boolean): boolean {
    if (isBookOwner) {
        return true;
    }

    if (!currentMemberId) {
        return false;
    }

    return settlement.createdById === currentMemberId || settlement.fromId === currentMemberId || settlement.toId === currentMemberId;
}

export function buildSettlementItems(settlements: BookSettlementInfoResponse[], members: BookSettleUpMemberResponse[], currentUsername: string | undefined, isBookOwner: boolean, unknownMemberName: string): SettlementItem[] {
    const memberMap = getSettleUpMemberMap(members);
    const currentMember = currentUsername ? members.find(member => member.username === currentUsername) : undefined;

    return settlements.map(settlement => ({
        id: settlement.id,
        fromName: getSettleUpMemberName(memberMap[settlement.fromId], unknownMemberName),
        toName: getSettleUpMemberName(memberMap[settlement.toId], unknownMemberName),
        amount: settlement.amount,
        currency: settlement.currency,
        time: settlement.time,
        comment: settlement.comment,
        canDelete: canDeleteSettlement(settlement, currentMember?.id, isBookOwner)
    }));
}
