import { useTransactionsStore } from '@/stores/transaction.ts';
import { useTransactionTemplatesStore } from '@/stores/transactionTemplate.ts';

import { KnownErrorCode } from '@/consts/api.ts';
import { TransactionType } from '@/core/transaction.ts';

import { Transaction } from '@/models/transaction.ts';
import type { TransactionTemplate } from '@/models/transaction_template.ts';

import services from '@/lib/services.ts';
import { generateRandomUUID } from '@/lib/misc.ts';
import { handleApiResponse } from '@/lib/api.ts';

export const SCHEDULED_CHANGE_NOT_APPLIED_MESSAGE: string = 'The transaction has been saved, but the changes could not be applied to its scheduled transaction';
export const SCHEDULED_CHANGE_CONFIRM_MESSAGE: string = 'This transaction was created by a scheduled transaction. Do you also want to apply these changes to the scheduled transaction and all its entries after this one? Choose OK to update them too, or Cancel to change only this transaction.';

function sameTagIds(tagIds1: string[], tagIds2: string[]): boolean {
    if (tagIds1.length !== tagIds2.length) {
        return false;
    }

    const sorted1 = [...tagIds1].sort();
    const sorted2 = [...tagIds2].sort();

    return sorted1.every((tagId, index) => tagId === sorted2[index]);
}

// the fields a scheduled transaction passes on to the transactions it creates
function copyScheduledFields(from: Transaction, to: Transaction | TransactionTemplate): void {
    to.type = from.type;
    to.expenseCategoryId = from.expenseCategoryId;
    to.incomeCategoryId = from.incomeCategoryId;
    to.transferCategoryId = from.transferCategoryId;
    to.sourceAccountId = from.sourceAccountId;
    to.destinationAccountId = from.destinationAccountId;
    to.sourceAmount = from.sourceAmount;
    to.destinationAmount = from.destinationAmount;
    to.hideAmount = from.hideAmount;
    to.tagIds = [...from.tagIds];
    to.comment = from.comment;
    to.merchant = from.merchant;
}

// lets the changes of a transaction created by a scheduled transaction also update the schedule and its later entries
export function useScheduledChangeBase() {
    const transactionsStore = useTransactionsStore();
    const transactionTemplatesStore = useTransactionTemplatesStore();

    function hasScheduledFieldChanges(original: Transaction, current: Transaction): boolean {
        if (!original.sourceTemplateId) {
            return false;
        }

        return original.type !== current.type
            || original.getCategoryId() !== current.getCategoryId()
            || original.sourceAccountId !== current.sourceAccountId
            || original.sourceAmount !== current.sourceAmount
            || (current.type === TransactionType.Transfer && (original.destinationAccountId !== current.destinationAccountId || original.destinationAmount !== current.destinationAmount))
            || original.hideAmount !== current.hideAmount
            || !sameTagIds(original.tagIds, current.tagIds)
            || original.comment !== current.comment
            || original.merchant !== current.merchant;
    }

    // resolves the number of later entries which were updated
    function applyToScheduleAndLaterTransactions(saved: Transaction, templateId: string, defaultCurrency: string): Promise<number> {
        return transactionTemplatesStore.getTemplate({ templateId }).then(template => {
            copyScheduledFields(saved, template);

            return transactionTemplatesStore.saveTemplateContent({
                template: template,
                isEdit: true,
                clientSessionId: generateRandomUUID()
            }).catch(error => {
                // the schedule may already have the same values
                if (error && error.error && error.error.errorCode === KnownErrorCode.NothingWillBeUpdated) {
                    return template;
                }

                return Promise.reject(error);
            });
        }).then(() => {
            return handleApiResponse(services.getLaterScheduledTransactionIds({ id: saved.id }), 'failed to retrieve later scheduled transactions', 'Unable to retrieve transaction list');
        }).then(transactionIds => {
            let promise: Promise<number> = Promise.resolve(0);

            // one by one, so the account balances are updated the same way as editing each of them
            for (const transactionId of transactionIds) {
                promise = promise.then(count => transactionsStore.getTransaction({ transactionId }).then(transaction => {
                    copyScheduledFields(saved, transaction);

                    return transactionsStore.saveTransaction({
                        transaction: transaction,
                        defaultCurrency: defaultCurrency,
                        isEdit: true,
                        clientSessionId: generateRandomUUID()
                    }).then(() => count + 1).catch(error => {
                        if (error && error.error && error.error.errorCode === KnownErrorCode.NothingWillBeUpdated) {
                            return count;
                        }

                        return Promise.reject(error);
                    });
                }));
            }

            return promise;
        });
    }

    return {
        hasScheduledFieldChanges,
        applyToScheduleAndLaterTransactions
    };
}
