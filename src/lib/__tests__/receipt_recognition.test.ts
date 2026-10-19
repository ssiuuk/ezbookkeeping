import { describe, expect, test } from 'vitest';

import { TransactionType } from '@/core/transaction.ts';
import type { RecognizedTransactionResponse } from '@/models/large_language_model.ts';
import { Transaction } from '@/models/transaction.ts';
import {
    getSetTransactionOptionsFromRecognizedResponse,
    getRecognizedTransactionQueryParams,
    shouldAttachRecognizedImageAsPicture,
    isPictureAttached
} from '@/lib/transaction.ts';

const fullResponse: RecognizedTransactionResponse = {
    type: TransactionType.Expense,
    time: 1760780096,
    categoryId: '201',
    sourceAccountId: '101',
    sourceAmount: 2345,
    tagIds: ['401', '402'],
    comment: 'milk & bread',
    merchant: 'Corner Market #12'
};

describe('getSetTransactionOptionsFromRecognizedResponse', () => {
    test('fills amount, time, merchant, category and account', () => {
        expect(getSetTransactionOptionsFromRecognizedResponse(fullResponse)).toEqual({
            type: TransactionType.Expense,
            time: 1760780096,
            categoryId: '201',
            accountId: '101',
            destinationAccountId: undefined,
            amount: 2345,
            destinationAmount: undefined,
            tagIds: '401,402',
            comment: 'milk & bread',
            merchant: 'Corner Market #12'
        });
    });

    test('leaves missing fields empty', () => {
        const options = getSetTransactionOptionsFromRecognizedResponse({ type: TransactionType.Expense });

        expect(options.type).toBe(TransactionType.Expense);
        expect(options.tagIds).toBeUndefined();
        expect(options.merchant).toBeUndefined();
        expect(options.categoryId).toBeUndefined();
    });
});

describe('getRecognizedTransactionQueryParams', () => {
    test('encodes every recognized field including merchant', () => {
        expect(getRecognizedTransactionQueryParams(fullResponse)).toEqual([
            `type=${TransactionType.Expense}`,
            'time=1760780096',
            'categoryId=201',
            'accountId=101',
            'amount=2345',
            'tagIds=401,402',
            'comment=milk%20%26%20bread',
            'merchant=Corner%20Market%20%2312'
        ]);
    });

    test('skips fields that were not recognized', () => {
        expect(getRecognizedTransactionQueryParams({ type: TransactionType.Transfer, sourceAmount: 100, destinationAmount: 90, destinationAccountId: '7', tagIds: [] })).toEqual([
            `type=${TransactionType.Transfer}`,
            'destinationAccountId=7',
            'amount=100',
            'destinationAmount=90'
        ]);
    });
});

describe('shouldAttachRecognizedImageAsPicture', () => {
    test('attaches only when transaction pictures are enabled and the setting is on', () => {
        expect(shouldAttachRecognizedImageAsPicture(true, true)).toBe(true);
        expect(shouldAttachRecognizedImageAsPicture(false, true)).toBe(false);
        expect(shouldAttachRecognizedImageAsPicture(true, false)).toBe(false);
        expect(shouldAttachRecognizedImageAsPicture(false, false)).toBe(false);
    });
});

describe('isPictureAttached', () => {
    test('finds the picture uploaded from the receipt image', () => {
        const transaction = Transaction.createNewTransaction(TransactionType.Expense, 1760780096, 'UTC', 0);

        expect(isPictureAttached(transaction, '9')).toBe(false);

        transaction.addPicture({ pictureId: '9', originalUrl: '/pictures/9.jpg' });

        expect(isPictureAttached(transaction, '9')).toBe(true);
        expect(isPictureAttached(transaction, '10')).toBe(false);
        expect(isPictureAttached(transaction, null)).toBe(false);

        transaction.clearPictures();

        expect(isPictureAttached(transaction, '9')).toBe(false);
    });
});
