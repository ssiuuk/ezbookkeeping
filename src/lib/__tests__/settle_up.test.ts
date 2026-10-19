import { describe, expect, test } from 'vitest';

import type { BookSettleUpResponse, BookSettlementInfoResponse } from '@/models/book.ts';

import { buildSettleUpCurrencyItems, buildSettlementItems, canDeleteSettlement } from '@/lib/settle_up.ts';

const settleUp: BookSettleUpResponse = {
    defaultCurrency: 'USD',
    startTime: 0,
    endTime: 0,
    members: [
        { id: '1', username: 'owner', nickname: 'Alice', isOwner: true, isFormer: false },
        { id: '2', username: 'bob', nickname: '', isOwner: false, isFormer: false },
        { id: '3', username: 'carol', nickname: 'Carol', isOwner: false, isFormer: true }
    ],
    currencies: [
        {
            currency: 'USD',
            totalExpense: 9000,
            balances: [
                { memberId: '1', paid: 6000, fairShare: 3000, settlementsSent: 0, settlementsReceived: 1000, balance: 2000 },
                { memberId: '2', paid: 0, fairShare: 3000, settlementsSent: 1000, settlementsReceived: 0, balance: -2000 },
                { memberId: '3', paid: 3000, fairShare: 3000, settlementsSent: 0, settlementsReceived: 0, balance: 0 }
            ],
            suggestedPayments: [
                { fromId: '2', toId: '1', amount: 2000 }
            ]
        },
        {
            currency: 'EUR',
            totalExpense: 0,
            balances: [
                { memberId: '1', paid: 0, fairShare: 0, settlementsSent: 0, settlementsReceived: 0, balance: 0 },
                { memberId: '9', paid: 0, fairShare: 0, settlementsSent: 0, settlementsReceived: 0, balance: 0 }
            ],
            suggestedPayments: []
        }
    ]
};

function settlement(id: string, fromId: string, toId: string, createdById: string): BookSettlementInfoResponse {
    return { id, fromId, toId, amount: 100, currency: 'USD', time: 1700000000, createdById, comment: '', createdAt: 1700000000 };
}

describe('buildSettleUpCurrencyItems', () => {
    test('keeps the currency order and names the members', () => {
        const items = buildSettleUpCurrencyItems(settleUp, 'Unknown');

        expect(items.map(item => item.currency)).toEqual(['USD', 'EUR']);
        expect(items[0]!.balances.map(balance => balance.memberName)).toEqual(['Alice', 'bob', 'Carol']);
        expect(items[0]!.balances[2]!.isFormer).toBe(true);
        expect(items[1]!.balances[1]!.memberName).toBe('Unknown');
    });

    test('nets the settlements and names the suggested payments', () => {
        const items = buildSettleUpCurrencyItems(settleUp, 'Unknown');

        expect(items[0]!.balances[0]!.settled).toBe(-1000);
        expect(items[0]!.balances[1]!.settled).toBe(1000);
        expect(items[0]!.settled).toBe(false);
        expect(items[0]!.payments).toEqual([
            { fromId: '2', fromName: 'bob', toId: '1', toName: 'Alice', amount: 2000, currency: 'USD' }
        ]);
        expect(items[1]!.settled).toBe(true);
    });
});

describe('canDeleteSettlement', () => {
    test('allows the owner and the members who took part', () => {
        expect(canDeleteSettlement(settlement('a', '2', '1', '2'), '3', true)).toBe(true);
        expect(canDeleteSettlement(settlement('a', '2', '1', '3'), '3', false)).toBe(true);
        expect(canDeleteSettlement(settlement('a', '2', '1', '1'), '2', false)).toBe(true);
        expect(canDeleteSettlement(settlement('a', '2', '1', '2'), '1', false)).toBe(true);
        expect(canDeleteSettlement(settlement('a', '2', '1', '2'), '3', false)).toBe(false);
        expect(canDeleteSettlement(settlement('a', '2', '1', '2'), undefined, false)).toBe(false);
    });
});

describe('buildSettlementItems', () => {
    test('finds the current member by username', () => {
        const items = buildSettlementItems([settlement('a', '2', '1', '1'), settlement('b', '1', '3', '1')], settleUp.members, 'bob', false, 'Unknown');

        expect(items.map(item => [item.fromName, item.toName, item.canDelete])).toEqual([
            ['bob', 'Alice', true],
            ['Alice', 'Carol', false]
        ]);
    });
});
