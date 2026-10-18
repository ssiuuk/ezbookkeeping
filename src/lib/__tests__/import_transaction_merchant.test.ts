import { describe, expect, test } from 'vitest';

import { TransactionType } from '@/core/transaction.ts';
import { ImportTransactionColumnType } from '@/core/import_transaction.ts';
import { KNOWN_COLUMN_NAME_MAPPING } from '@/consts/import_transaction.ts';
import { TRANSACTION_MAX_MERCHANT_LENGTH } from '@/consts/transaction.ts';
import { ImportTransaction, type ImportTransactionResponse } from '@/models/imported_transaction.ts';
import { Transaction } from '@/models/transaction.ts';
import { setTransactionModelByTransaction } from '@/lib/transaction.ts';

function createImportTransactionResponse(merchant?: string): ImportTransactionResponse {
    return {
        type: TransactionType.Expense,
        categoryId: '1',
        originalCategoryName: 'Test Category',
        time: 1725165296,
        utcOffset: 480,
        sourceAccountId: '2',
        originalSourceAccountName: 'Test Account',
        originalSourceAccountCurrency: 'CNY',
        sourceAmount: 12345,
        tagIds: [],
        originalTagNames: [],
        comment: 'Coffee',
        merchant: merchant
    };
}

describe('ImportTransaction merchant', () => {
    test('reads merchant from the parsed response and sends it in the create request', () => {
        const transaction = ImportTransaction.of(createImportTransactionResponse('  Test Shop '), 0);

        expect(transaction.merchant).toBe('  Test Shop ');
        expect(transaction.valid).toBe(true);
        expect(transaction.toCreateRequest().merchant).toBe('Test Shop');
        expect(transaction.toCreateRequest().comment).toBe('Coffee');
    });

    test('defaults to empty merchant when the response has none', () => {
        const transaction = ImportTransaction.of(createImportTransactionResponse(), 0);

        expect(transaction.merchant).toBe('');
        expect(transaction.toCreateRequest().merchant).toBe('');
    });

    test('marks the transaction invalid when merchant is too long', () => {
        const transaction = ImportTransaction.of(createImportTransactionResponse('a'.repeat(TRANSACTION_MAX_MERCHANT_LENGTH + 1)), 0);

        expect(transaction.valid).toBe(false);

        transaction.merchant = 'a'.repeat(TRANSACTION_MAX_MERCHANT_LENGTH);
        expect(transaction.isTransactionValid()).toBe(true);
    });
});

describe('import column name mapping', () => {
    test('maps the merchant column of ezbookkeeping export and common merchant column names', () => {
        expect(ImportTransactionColumnType.Merchant.type).toBe(104);
        expect(KNOWN_COLUMN_NAME_MAPPING['merchant']).toBe(ImportTransactionColumnType.Merchant);
        expect(KNOWN_COLUMN_NAME_MAPPING['商家']).toBe(ImportTransactionColumnType.Merchant);
        expect(KNOWN_COLUMN_NAME_MAPPING['交易对方']).toBe(ImportTransactionColumnType.Merchant);
        expect(KNOWN_COLUMN_NAME_MAPPING['payee']).toBe(ImportTransactionColumnType.Merchant);
        expect(KNOWN_COLUMN_NAME_MAPPING['description']).toBe(ImportTransactionColumnType.Description);
    });
});

describe('setTransactionModelByTransaction merchant option', () => {
    test('applies the recognized merchant', () => {
        const transaction = Transaction.createNewTransaction(TransactionType.Expense, 1725165296, '', 480);

        setTransactionModelByTransaction(transaction, null, {}, {}, [], {}, {}, '', { comment: 'Coffee', merchant: ' Test Shop ' }, false);

        expect(transaction.comment).toBe('Coffee');
        expect(transaction.merchant).toBe('Test Shop');
    });

    test('keeps the current merchant when the recognized merchant is empty', () => {
        const transaction = Transaction.createNewTransaction(TransactionType.Expense, 1725165296, '', 480);
        transaction.merchant = 'Old Shop';

        setTransactionModelByTransaction(transaction, null, {}, {}, [], {}, {}, '', { merchant: '  ' }, false);

        expect(transaction.merchant).toBe('Old Shop');
    });
});
