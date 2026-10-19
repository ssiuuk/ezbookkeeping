import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useUserStore } from '@/stores/user.ts';
import { useBooksStore } from '@/stores/book.ts';

import type { BookInfoResponse, BookSettleUpResponse, BookSettlementInfoResponse } from '@/models/book.ts';

import {
    type SettleUpCurrencyItem,
    type SettleUpPaymentItem,
    type SettlementItem,
    buildSettleUpCurrencyItems,
    buildSettlementItems
} from '@/lib/settle_up.ts';
import { parseDateTimeFromUnixTime } from '@/lib/datetime.ts';
import { parseBigDecimal } from '@/lib/numeral.ts';
import { handleApiResponse } from '@/lib/api.ts';
import services from '@/lib/services.ts';

export function useSettleUpPageBase(bookId: string) {
    const { tt, formatAmountToLocalizedNumeralsWithCurrency, formatDateTimeToLongDateTime } = useI18n();

    const userStore = useUserStore();
    const booksStore = useBooksStore();

    const loading = ref<boolean>(true);
    const book = ref<BookInfoResponse | undefined>(undefined);
    const settleUp = ref<BookSettleUpResponse | undefined>(undefined);
    const settlements = ref<BookSettlementInfoResponse[]>([]);

    const options = { bookId: bookId };

    const bookName = computed<string>(() => book.value?.name ?? '');
    const currencyItems = computed<SettleUpCurrencyItem[]>(() => settleUp.value ? buildSettleUpCurrencyItems(settleUp.value, tt('Unknown')) : []);
    const settlementItems = computed<SettlementItem[]>(() => buildSettlementItems(settlements.value, settleUp.value?.members ?? [], userStore.currentUserBasicInfo?.username, !!book.value?.isOwner, tt('Unknown')));

    function reload(): Promise<void> {
        loading.value = true;

        return Promise.all([
            booksStore.loadAllBooks(),
            handleApiResponse(services.getBookSettleUp({}, options), 'failed to load settle up', 'Unable to retrieve settle up data'),
            handleApiResponse(services.getBookSettlements({}, options), 'failed to load settlements', 'Unable to retrieve settlement list')
        ]).then(([books, settleUpResult, settlementsResult]) => {
            book.value = books.find(item => item.id === bookId && !item.isPersonal);
            settleUp.value = settleUpResult;
            settlements.value = settlementsResult;
            loading.value = false;
        }).catch(error => {
            loading.value = false;
            throw error;
        });
    }

    function markAsSettled(payment: SettleUpPaymentItem): Promise<void> {
        return handleApiResponse(services.addBookSettlement({
            fromId: payment.fromId,
            toId: payment.toId,
            amount: payment.amount,
            currency: payment.currency
        }, options), 'failed to add settlement', 'Unable to record settlement').then(() => reload());
    }

    function deleteSettlement(settlement: SettlementItem): Promise<void> {
        return handleApiResponse(services.deleteBookSettlement({ id: settlement.id }, options), 'failed to delete settlement', 'Unable to delete settlement').then(() => reload());
    }

    function formatAmount(amount: number, currency: string): string {
        return formatAmountToLocalizedNumeralsWithCurrency(parseBigDecimal(amount), currency);
    }

    function formatTime(unixTime: number): string {
        return formatDateTimeToLongDateTime(parseDateTimeFromUnixTime(unixTime));
    }

    return {
        // states
        loading,
        // computed states
        bookName,
        currencyItems,
        settlementItems,
        // functions
        reload,
        markAsSettled,
        deleteSettlement,
        formatAmount,
        formatTime
    };
}
