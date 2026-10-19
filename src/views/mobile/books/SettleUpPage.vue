<template>
    <f7-page ptr @ptr:refresh="reloadData" @page:afterin="onPageAfterIn">
        <f7-navbar :title="tt('Settle Up')" :subtitle="bookName" :back-link="tt('Back')"></f7-navbar>

        <f7-block class="margin-top-half margin-bottom-half">
            <small>{{ tt('Each expense is paid by the member who owns the account it was paid from, and is split equally among the current members and anyone else who paid. Expenses paid from accounts of the book itself, income and transfers are left out. Each currency is settled up on its own.') }}</small>
        </f7-block>

        <f7-list strong inset dividers v-if="!loading && !currencyItems.length">
            <f7-list-item :title="tt('No expense to settle up yet')"></f7-list-item>
        </f7-list>

        <template :key="item.currency" v-for="item in currencyItems">
            <f7-block-title>{{ item.currency }} · {{ tt('Total Expense') }}: {{ formatAmount(item.totalExpense, item.currency) }}</f7-block-title>
            <f7-list strong inset dividers :class="{ 'skeleton-text': loading }">
                <f7-list-item :key="balance.memberId"
                              :footer="`${tt('Paid')}: ${formatAmount(balance.paid, item.currency)} · ${tt('Fair Share')}: ${formatAmount(balance.fairShare, item.currency)} · ${tt('Settled')}: ${formatAmount(balance.settled, item.currency)}`"
                              v-for="balance in item.balances">
                    <template #title>
                        <span>{{ balance.memberName }}</span>
                        <f7-badge class="margin-left-half" v-if="balance.isFormer">{{ tt('Former Member') }}</f7-badge>
                    </template>
                    <template #after>
                        <span :class="{ 'text-color-teal': balance.balance > 0, 'text-color-red': balance.balance < 0 }">
                            {{ balance.balance > 0 ? tt('Is owed') : (balance.balance < 0 ? tt('Owes') : tt('Settled')) }}
                            <span v-if="balance.balance !== 0">{{ formatAmount(Math.abs(balance.balance), item.currency) }}</span>
                        </span>
                    </template>
                </f7-list-item>
            </f7-list>

            <f7-block-header>{{ tt('Suggested Payments') }}</f7-block-header>
            <f7-list strong inset dividers>
                <f7-list-item :title="tt('Everyone is settled up')" v-if="item.settled"></f7-list-item>
                <f7-list-item :key="`${payment.fromId}-${payment.toId}`"
                              :title="tt('{from} pays {to} {amount}', { from: payment.fromName, to: payment.toName, amount: formatAmount(payment.amount, payment.currency) })"
                              v-for="payment in item.payments">
                    <template #after>
                        <f7-link :class="{ 'disabled': loading }" @click="settle(payment)">{{ tt('Mark as Settled') }}</f7-link>
                    </template>
                </f7-list-item>
            </f7-list>
        </template>

        <f7-block-title>{{ tt('Settlement History') }}</f7-block-title>
        <f7-list strong inset dividers :class="{ 'skeleton-text': loading }">
            <f7-list-item :title="tt('No settlement yet')" v-if="!settlementItems.length"></f7-list-item>
            <f7-list-item :key="settlement.id"
                          :title="tt('{from} paid {to} {amount}', { from: settlement.fromName, to: settlement.toName, amount: formatAmount(settlement.amount, settlement.currency) })"
                          :footer="settlement.comment ? `${formatTime(settlement.time)} · ${settlement.comment}` : formatTime(settlement.time)"
                          v-for="settlement in settlementItems">
                <template #after>
                    <f7-link color="red" :class="{ 'disabled': loading }" v-if="settlement.canDelete" @click="remove(settlement)">{{ tt('Delete') }}</f7-link>
                </template>
            </f7-list-item>
        </f7-list>
    </f7-page>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading } from '@/lib/ui/mobile.ts';
import { useSettleUpPageBase } from '@/views/base/books/SettleUpPageBase.ts';

import type { SettleUpPaymentItem, SettlementItem } from '@/lib/settle_up.ts';

const props = defineProps<{
    f7route: Router.Route;
    f7router: Router.Router;
}>();

const { tt } = useI18n();
const { showConfirm, showToast, routeBackOnError } = useI18nUIComponents();
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
} = useSettleUpPageBase((props.f7route.query['bookId'] as string | undefined) ?? '');

const loadingError = ref<unknown | null>(null);

function runWithLoading(promise: () => Promise<void>, successMessage: string): void {
    showLoading();

    promise().then(() => {
        hideLoading();
        showToast(successMessage);
    }).catch(error => {
        hideLoading();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function reloadData(done?: () => void): void {
    reload().then(() => {
        done?.();
    }).catch(error => {
        done?.();

        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function settle(payment: SettleUpPaymentItem): void {
    showConfirm('Are you sure you want to mark this payment as settled?', () => {
        runWithLoading(() => markAsSettled(payment), 'You have recorded this settlement');
    });
}

function remove(settlement: SettlementItem): void {
    showConfirm('Are you sure you want to delete this settlement?', () => {
        runWithLoading(() => deleteSettlement(settlement), 'You have deleted this settlement');
    });
}

function onPageAfterIn(): void {
    routeBackOnError(props.f7router, loadingError);
}

reload().catch(error => {
    if (!error.processed) {
        loadingError.value = error;
        showToast(error.message || error);
    }
});
</script>
