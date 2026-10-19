import { describe, expect, test } from 'vitest';

import type { BudgetAlertInfoResponse } from '@/models/budget_alert.ts';
import { getBudgetAlertNoticeItems, getBudgetAlertNoticeItemIds } from '@/lib/budget_alert.ts';

function alert(id: string, categoryId: string, period: string, threshold: number, time: number, dismissed?: boolean): BudgetAlertInfoResponse {
    return {
        id: id,
        categoryId: categoryId,
        categoryName: `Category ${categoryId}`,
        period: period,
        threshold: threshold,
        spent: threshold * 100,
        budget: 10000,
        currency: 'USD',
        dismissed: !!dismissed,
        time: time
    };
}

describe('getBudgetAlertNoticeItems', () => {
    test('returns nothing without alerts', () => {
        expect(getBudgetAlertNoticeItems([])).toEqual([]);
    });

    test('leaves out dismissed alerts', () => {
        const items = getBudgetAlertNoticeItems([alert('1', 'a', '2026-10', 80, 100, true), alert('2', 'b', '2026-10', 80, 200)]);

        expect(items.map(item => item.alert.id)).toEqual(['2']);
        expect(items[0]?.exceeded).toBe(false);
    });

    test('shows only the highest threshold of a category in a period and dismisses them together', () => {
        const items = getBudgetAlertNoticeItems([alert('1', 'a', '2026-10', 80, 100), alert('2', 'a', '2026-10', 100, 200)]);

        expect(items).toHaveLength(1);
        expect(items[0]?.alert.id).toBe('2');
        expect(items[0]?.exceeded).toBe(true);
        expect(items[0]?.alertIds).toEqual(['1', '2']);
    });

    test('keeps alerts of different periods and categories apart, latest first', () => {
        const items = getBudgetAlertNoticeItems([
            alert('1', 'a', '2026-09', 100, 100),
            alert('2', 'a', '2026-10', 80, 300),
            alert('3', 'b', '2026-10', 80, 200)
        ]);

        expect(items.map(item => item.alert.id)).toEqual(['2', '3', '1']);
    });

    test('orders alerts of the same time by threshold and limits the count', () => {
        const items = getBudgetAlertNoticeItems([
            alert('1', 'a', '2026-10', 80, 100),
            alert('2', 'b', '2026-10', 100, 100),
            alert('3', 'c', '2026-10', 80, 50)
        ], 2);

        expect(items.map(item => item.alert.id)).toEqual(['2', '1']);
    });
});

describe('getBudgetAlertNoticeItemIds', () => {
    test('returns the ids of all alerts of the notices', () => {
        const items = getBudgetAlertNoticeItems([alert('1', 'a', '2026-10', 80, 100), alert('2', 'a', '2026-10', 100, 200), alert('3', 'b', '2026-10', 80, 50)]);

        expect(getBudgetAlertNoticeItemIds(items)).toEqual(['1', '2', '3']);
    });
});
