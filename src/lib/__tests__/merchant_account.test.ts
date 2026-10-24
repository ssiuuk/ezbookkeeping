import { describe, expect, test } from 'vitest';

import { MerchantAccountLookup, findKnownMerchant } from '@/lib/merchant_account.ts';

interface Deferred {
    readonly merchant: string;
    readonly resolve: (accountId: string | undefined) => void;
}

function createLookup(): { lookup: MerchantAccountLookup, requests: Deferred[] } {
    const requests: Deferred[] = [];
    const lookup = new MerchantAccountLookup(merchant => new Promise<string | undefined>(resolve => {
        requests.push({ merchant, resolve });
    }));

    return { lookup, requests };
}

const knownMerchants = ['Cafe', 'Costco Wholesale'];

describe('findKnownMerchant', () => {
    test('matches a used shop ignoring case and spaces', () => {
        expect(findKnownMerchant(knownMerchants, ' costco wholesale ')).toBe('Costco Wholesale');
    });

    test('returns nothing for an empty or unknown shop', () => {
        expect(findKnownMerchant(knownMerchants, '')).toBeUndefined();
        expect(findKnownMerchant(knownMerchants, 'Bakery')).toBeUndefined();
    });
});

describe('MerchantAccountLookup', () => {
    test('applies the account of a known shop', async () => {
        const { lookup, requests } = createLookup();
        const applied: string[] = [];
        const done = lookup.lookup({ merchant: 'cafe', knownMerchants, currentMerchant: () => 'Cafe', isStillCurrent: () => true, apply: id => applied.push(id) });

        expect(requests.map(request => request.merchant)).toEqual(['Cafe']);
        requests[0]!.resolve('101');
        await done;
        expect(applied).toEqual(['101']);
    });

    test('drops the answer when the shop is cleared or changed to an unknown one before it arrives', async () => {
        const { lookup, requests } = createLookup();
        const applied: string[] = [];
        let merchant = 'Cafe';
        const first = lookup.lookup({ merchant, knownMerchants, currentMerchant: () => merchant, isStillCurrent: () => true, apply: id => applied.push(id) });

        merchant = '';
        await lookup.lookup({ merchant, knownMerchants, currentMerchant: () => merchant, isStillCurrent: () => true, apply: id => applied.push(id) });
        merchant = 'Bakery';
        await lookup.lookup({ merchant, knownMerchants, currentMerchant: () => merchant, isStillCurrent: () => true, apply: id => applied.push(id) });

        requests[0]!.resolve('101');
        await first;
        expect(requests.length).toBe(1);
        expect(applied).toEqual([]);
    });

    test('drops the answer when the shop in the editor no longer matches', async () => {
        const { lookup, requests } = createLookup();
        const applied: string[] = [];
        let merchant = 'Cafe';
        const done = lookup.lookup({ merchant, knownMerchants, currentMerchant: () => merchant, isStillCurrent: () => true, apply: id => applied.push(id) });

        merchant = 'Cafe Nero';
        requests[0]!.resolve('101');
        await done;
        expect(applied).toEqual([]);
    });

    test('drops the answer after the editor is closed and opened again', async () => {
        const { lookup, requests } = createLookup();
        const applied: string[] = [];
        const done = lookup.lookup({ merchant: 'Cafe', knownMerchants, currentMerchant: () => 'Cafe', isStillCurrent: () => true, apply: id => applied.push(id) });

        lookup.cancel();
        requests[0]!.resolve('101');
        await done;
        expect(applied).toEqual([]);
    });

    test('drops the answer when the editor is no longer adding or the account was picked by hand', async () => {
        const { lookup, requests } = createLookup();
        const applied: string[] = [];
        const done = lookup.lookup({ merchant: 'Cafe', knownMerchants, currentMerchant: () => 'Cafe', isStillCurrent: () => false, apply: id => applied.push(id) });

        requests[0]!.resolve('101');
        await done;
        expect(applied).toEqual([]);
    });

    test('only the latest of two lookups is applied, and an empty answer changes nothing', async () => {
        const { lookup, requests } = createLookup();
        const applied: string[] = [];
        let merchant = 'Cafe';
        const first = lookup.lookup({ merchant, knownMerchants, currentMerchant: () => merchant, isStillCurrent: () => true, apply: id => applied.push(id) });
        merchant = 'Costco Wholesale';
        const second = lookup.lookup({ merchant, knownMerchants, currentMerchant: () => merchant, isStillCurrent: () => true, apply: id => applied.push(id) });

        requests[1]!.resolve('202');
        requests[0]!.resolve('101');
        await Promise.all([first, second]);
        expect(applied).toEqual(['202']);

        merchant = 'Cafe';
        const third = lookup.lookup({ merchant, knownMerchants, currentMerchant: () => merchant, isStillCurrent: () => true, apply: id => applied.push(id) });
        requests[2]!.resolve('0');
        await third;
        expect(applied).toEqual(['202']);
    });
});
