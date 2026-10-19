<template>
    <main-page-layout no-navbar>
        <template #content>
            <v-row align-content="start">
                <v-col cols="12">
                    <v-card>
                        <template #title>
                            <div class="d-flex align-center flex-wrap">
                                <span>{{ tt('Settle Up') }}</span>
                                <span class="text-medium-emphasis ms-2" v-if="bookName">{{ bookName }}</span>
                                <v-spacer/>
                                <v-btn density="comfortable" variant="text" :icon="mdiRefresh"
                                       :aria-label="tt('Refresh')" :disabled="loading" @click="reloadData"/>
                                <v-progress-circular indeterminate size="20" class="ms-3" v-if="loading"></v-progress-circular>
                            </div>
                        </template>
                        <v-card-text>
                            <div class="text-body-medium text-medium-emphasis">{{ tt('Each expense is paid by the member who owns the account it was paid from, and is split equally among the current members and anyone else who paid. Expenses paid from accounts of the book itself, income and transfers are left out. Each currency is settled up on its own.') }}</div>
                        </v-card-text>
                    </v-card>
                </v-col>

                <v-col cols="12" v-if="!loading && !currencyItems.length">
                    <v-card>
                        <v-card-text>{{ tt('No expense to settle up yet') }}</v-card-text>
                    </v-card>
                </v-col>

                <v-col cols="12" :key="item.currency" v-for="item in currencyItems">
                    <v-card :class="{ 'disabled': loading }">
                        <template #title>
                            <span>{{ item.currency }}</span>
                        </template>
                        <template #subtitle>
                            <span>{{ tt('Total Expense') }}: {{ formatAmount(item.totalExpense, item.currency) }}</span>
                        </template>

                        <v-table class="text-no-wrap">
                            <thead>
                            <tr>
                                <th>{{ tt('Member') }}</th>
                                <th class="text-right">{{ tt('Paid') }}</th>
                                <th class="text-right">{{ tt('Fair Share') }}</th>
                                <th class="text-right">{{ tt('Settled') }}</th>
                                <th class="text-right">{{ tt('Balance') }}</th>
                            </tr>
                            </thead>
                            <tbody>
                            <tr :key="balance.memberId" v-for="balance in item.balances">
                                <td>
                                    <span>{{ balance.memberName }}</span>
                                    <v-chip class="ms-2" size="x-small" v-if="balance.isFormer">{{ tt('Former Member') }}</v-chip>
                                </td>
                                <td class="text-right">{{ formatAmount(balance.paid, item.currency) }}</td>
                                <td class="text-right">{{ formatAmount(balance.fairShare, item.currency) }}</td>
                                <td class="text-right">{{ formatAmount(balance.settled, item.currency) }}</td>
                                <td class="text-right" :class="{ 'text-income': balance.balance > 0, 'text-expense': balance.balance < 0 }">
                                    {{ balance.balance > 0 ? tt('Is owed') : (balance.balance < 0 ? tt('Owes') : tt('Settled')) }}
                                    <span v-if="balance.balance !== 0">{{ formatAmount(Math.abs(balance.balance), item.currency) }}</span>
                                </td>
                            </tr>
                            </tbody>
                        </v-table>

                        <v-card-text>
                            <div class="text-title-medium mb-2">{{ tt('Suggested Payments') }}</div>
                            <div class="text-body-medium" v-if="item.settled">{{ tt('Everyone is settled up') }}</div>
                            <div class="d-flex align-center flex-wrap py-1"
                                 :key="`${payment.fromId}-${payment.toId}`" v-for="payment in item.payments">
                                <span class="text-body-large">{{ tt('{from} pays {to} {amount}', { from: payment.fromName, to: payment.toName, amount: formatAmount(payment.amount, payment.currency) }) }}</span>
                                <v-spacer/>
                                <v-btn density="comfortable" variant="tonal" :disabled="loading || submitting"
                                       @click="settle(payment)">
                                    {{ tt('Mark as Settled') }}
                                </v-btn>
                            </div>
                        </v-card-text>
                    </v-card>
                </v-col>

                <v-col cols="12">
                    <v-card :title="tt('Settlement History')" :class="{ 'disabled': loading }">
                        <v-card-text v-if="!settlementItems.length">{{ tt('No settlement yet') }}</v-card-text>
                        <v-table class="text-no-wrap" v-else>
                            <tbody>
                            <tr :key="settlement.id" v-for="settlement in settlementItems">
                                <td>{{ formatTime(settlement.time) }}</td>
                                <td>{{ tt('{from} paid {to} {amount}', { from: settlement.fromName, to: settlement.toName, amount: formatAmount(settlement.amount, settlement.currency) }) }}</td>
                                <td>{{ settlement.comment }}</td>
                                <td class="text-right">
                                    <v-btn density="comfortable" color="error" variant="text"
                                           :disabled="loading || submitting"
                                           v-if="settlement.canDelete" @click="remove(settlement)">
                                        {{ tt('Delete') }}
                                    </v-btn>
                                </td>
                            </tr>
                            </tbody>
                        </v-table>
                    </v-card>
                </v-col>
            </v-row>
        </template>
    </main-page-layout>

    <confirm-dialog ref="confirmDialog"/>
    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import SnackBar from '@/components/desktop/SnackBar.vue';

import { ref, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useSettleUpPageBase } from '@/views/base/books/SettleUpPageBase.ts';

import type { SettleUpPaymentItem, SettlementItem } from '@/lib/settle_up.ts';

import {
    mdiRefresh
} from '@mdi/js';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type SnackBarType = InstanceType<typeof SnackBar>;

const props = defineProps<{
    initBookId?: string;
}>();

const { tt } = useI18n();
const {
    loading,
    bookName,
    currencyItems,
    settlementItems,
    reload,
    markAsSettled,
    deleteSettlement,
    formatAmount,
    formatTime
} = useSettleUpPageBase(props.initBookId || '');

const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const snackbar = useTemplateRef<SnackBarType>('snackbar');

const submitting = ref<boolean>(false);

function onError(error: { processed?: boolean, message?: string }): void {
    if (!error.processed) {
        snackbar.value?.showError(error as { message: string });
    }
}

function reloadData(): void {
    reload().catch(onError);
}

function settle(payment: SettleUpPaymentItem): void {
    confirmDialog.value?.open('Are you sure you want to mark this payment as settled?').then(() => {
        submitting.value = true;

        markAsSettled(payment).then(() => {
            submitting.value = false;
            snackbar.value?.showMessage('You have recorded this settlement');
        }).catch(error => {
            submitting.value = false;
            onError(error);
        });
    });
}

function remove(settlement: SettlementItem): void {
    confirmDialog.value?.open('Are you sure you want to delete this settlement?', { color: 'error' }).then(() => {
        submitting.value = true;

        deleteSettlement(settlement).then(() => {
            submitting.value = false;
            snackbar.value?.showMessage('You have deleted this settlement');
        }).catch(error => {
            submitting.value = false;
            onError(error);
        });
    });
}

reloadData();
</script>
