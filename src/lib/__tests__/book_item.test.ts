import { describe, expect, test } from 'vitest';

import { findSameBookItemId, findSameBookItemIds } from '@/lib/book_item.ts';

interface TestItem {
    readonly id: string;
    readonly name: string;
    readonly parent?: string;
    readonly hidden?: boolean;
}

const isVisible = (item: TestItem) => !item.hidden;
const getParentName = (item: TestItem) => item.parent;

describe('findSameBookItemId', () => {
    const items: TestItem[] = [
        { id: '11', name: 'Lunch', parent: 'Meals', hidden: true },
        { id: '12', name: 'Lunch', parent: 'Work' },
        { id: '13', name: 'Lunch', parent: 'Food' },
        { id: '14', name: 'Taxi', parent: 'Transport' }
    ];

    test('returns the referenced item itself when it is in the list', () => {
        expect(findSameBookItemId(items, { id: '11', name: 'Something Else' }, isVisible, getParentName)).toBe('11');
    });

    test('prefers the item under a parent with the same name, then a visible one', () => {
        expect(findSameBookItemId(items, { id: '1', name: 'Lunch', parentName: 'Food' }, isVisible, getParentName)).toBe('13');
        expect(findSameBookItemId(items, { id: '1', name: 'Lunch', parentName: 'Meals' }, isVisible, getParentName)).toBe('11');
        expect(findSameBookItemId(items, { id: '1', name: 'Lunch', parentName: 'Other' }, isVisible, getParentName)).toBe('12');
        expect(findSameBookItemId(items, { id: '1', name: 'Lunch' }, isVisible)).toBe('12');
    });

    test('ignores surrounding spaces in names', () => {
        expect(findSameBookItemId(items, { id: '1', name: ' Taxi ' }, isVisible)).toBe('14');
    });

    test('returns empty when no item has the same name', () => {
        expect(findSameBookItemId(items, { id: '1', name: 'Dinner' }, isVisible)).toBe('');
        expect(findSameBookItemId(items, { id: '1', name: '' }, isVisible)).toBe('');
        expect(findSameBookItemId(items, undefined, isVisible)).toBe('');
        expect(findSameBookItemId([], { id: '1', name: 'Lunch' }, isVisible)).toBe('');
    });
});

describe('findSameBookItemIds', () => {
    const tags: TestItem[] = [
        { id: '21', name: 'Trip' },
        { id: '22', name: 'Kids' },
        { id: '23', name: 'Kids', hidden: true }
    ];

    test('keeps the found items in order and skips the missing ones', () => {
        expect(findSameBookItemIds(tags, [
            { id: '1', name: 'Kids' },
            { id: '2', name: 'Gift' },
            { id: '3', name: 'Trip' }
        ], isVisible)).toEqual(['22', '21']);
    });

    test('does not return an item twice', () => {
        expect(findSameBookItemIds(tags, [
            { id: '21', name: 'Trip' },
            { id: '4', name: 'Trip' }
        ], isVisible)).toEqual(['21']);
        expect(findSameBookItemIds(tags, [], isVisible)).toEqual([]);
    });
});
