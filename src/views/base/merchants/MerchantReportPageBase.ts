import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useUserStore } from '@/stores/user.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';

import type { TransactionMerchantStatisticResponseItem } from '@/models/transaction.ts';

import { type MerchantReport, buildMerchantReport } from '@/lib/merchant_report.ts';
import {
    getCurrentUnixTime,
    getYear0BasedMonthObjectFromUnixTime,
    getYearMonthFirstUnixTime,
    getYearMonthLastUnixTime,
    parseDateTimeFromUnixTime
} from '@/lib/datetime.ts';
import { parseBigDecimal } from '@/lib/numeral.ts';
import { handleApiResponse } from '@/lib/api.ts';
import services from '@/lib/services.ts';

export type MerchantReportPeriodType = 'month' | 'year';

export function useMerchantReportPageBase() {
    const { tt, formatDateTimeToGregorianLikeLongYearMonth, formatDateTimeToGregorianLikeLongYear, formatAmountToLocalizedNumeralsWithCurrency, formatPercentToLocalizedNumerals } = useI18n();

    const userStore = useUserStore();
    const accountsStore = useAccountsStore();
    const exchangeRatesStore = useExchangeRatesStore();

    const now = getYear0BasedMonthObjectFromUnixTime(getCurrentUnixTime());

    const loading = ref<boolean>(true);
    const periodType = ref<MerchantReportPeriodType>('month');
    const year = ref<number>(now.year);
    const month0base = ref<number>(now.month0base);
    const statisticItems = ref<TransactionMerchantStatisticResponseItem[]>([]);

    const defaultCurrency = computed<string>(() => userStore.currentUserDefaultCurrency);

    const periodStartTime = computed<number>(() => getYearMonthFirstUnixTime({ year: year.value, month0base: periodType.value === 'year' ? 0 : month0base.value }));
    const periodEndTime = computed<number>(() => getYearMonthLastUnixTime({ year: year.value, month0base: periodType.value === 'year' ? 11 : month0base.value }));

    const periodDisplayName = computed<string>(() => {
        const dateTime = parseDateTimeFromUnixTime(periodStartTime.value);
        return periodType.value === 'year' ? formatDateTimeToGregorianLikeLongYear(dateTime) : formatDateTimeToGregorianLikeLongYearMonth(dateTime);
    });

    const report = computed<MerchantReport>(() => buildMerchantReport(statisticItems.value, accountsStore.allAccountsMap, (amount, currency) => {
        if (currency === defaultCurrency.value) {
            return amount;
        }

        const exchangedAmount = exchangeRatesStore.getExchangedAmount(parseBigDecimal(amount), currency, defaultCurrency.value);
        return exchangedAmount ? exchangedAmount.truncate().toDoubleNumber() : null;
    }));

    function loadReport(): Promise<void> {
        loading.value = true;

        return Promise.all([
            accountsStore.loadAllAccounts({ force: false }),
            handleApiResponse(services.getTransactionMerchantStatistics({
                startTime: periodStartTime.value,
                endTime: periodEndTime.value
            }), 'failed to retrieve merchant statistics', 'Unable to retrieve merchant report')
        ]).then(([, statistics]) => {
            statisticItems.value = statistics.items ?? [];
            loading.value = false;
        }).catch(error => {
            loading.value = false;
            throw error;
        });
    }

    function setPeriodType(type: MerchantReportPeriodType): Promise<void> {
        periodType.value = type;
        return loadReport();
    }

    function changePeriod(offset: number): Promise<void> {
        if (periodType.value === 'year') {
            year.value += offset;
        } else {
            let newMonth = month0base.value + offset;
            let newYear = year.value;

            while (newMonth < 0) {
                newMonth += 12;
                newYear--;
            }

            while (newMonth > 11) {
                newMonth -= 12;
                newYear++;
            }

            year.value = newYear;
            month0base.value = newMonth;
        }

        return loadReport();
    }

    function getMerchantDisplayName(merchant: string): string {
        return merchant || tt('(No merchant)');
    }

    function formatAmount(amount: number): string {
        return formatAmountToLocalizedNumeralsWithCurrency(parseBigDecimal(amount), defaultCurrency.value);
    }

    function formatPercent(value: number): string {
        return formatPercentToLocalizedNumerals(value, 1, '<0.1');
    }

    return {
        // states
        loading,
        periodType,
        // computed states
        periodDisplayName,
        report,
        // functions
        loadReport,
        setPeriodType,
        changePeriod,
        getMerchantDisplayName,
        formatAmount,
        formatPercent
    };
}
