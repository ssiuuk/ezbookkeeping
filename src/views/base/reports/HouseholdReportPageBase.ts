import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useUserStore } from '@/stores/user.ts';
import { useBooksStore } from '@/stores/book.ts';
import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';

import { KeywordMatchMode } from '@/core/text.ts';

import type { BookInfoResponse } from '@/models/book.ts';

import { useBookBase } from '@/views/base/books/BookBase.ts';

import { type HouseholdBookData, type HouseholdReport, buildHouseholdReport } from '@/lib/household_report.ts';
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

export type HouseholdReportPeriodType = 'month' | 'year';

export function useHouseholdReportPageBase() {
    const { tt, formatDateTimeToGregorianLikeLongYearMonth, formatDateTimeToGregorianLikeLongYear, formatAmountToLocalizedNumeralsWithCurrency, formatPercentToLocalizedNumerals } = useI18n();
    const { getBookDisplayName } = useBookBase();

    const userStore = useUserStore();
    const booksStore = useBooksStore();
    const exchangeRatesStore = useExchangeRatesStore();

    const now = getYear0BasedMonthObjectFromUnixTime(getCurrentUnixTime());

    const loading = ref<boolean>(true);
    const periodType = ref<HouseholdReportPeriodType>('month');
    const year = ref<number>(now.year);
    const month0base = ref<number>(now.month0base);
    const bookData = ref<HouseholdBookData[]>([]);
    const excludedBookIds = ref<Record<string, boolean>>({});

    const defaultCurrency = computed<string>(() => userStore.currentUserDefaultCurrency);

    const periodStartTime = computed<number>(() => getYearMonthFirstUnixTime({ year: year.value, month0base: periodType.value === 'year' ? 0 : month0base.value }));
    const periodEndTime = computed<number>(() => getYearMonthLastUnixTime({ year: year.value, month0base: periodType.value === 'year' ? 11 : month0base.value }));

    const periodDisplayName = computed<string>(() => {
        const dateTime = parseDateTimeFromUnixTime(periodStartTime.value);
        return periodType.value === 'year' ? formatDateTimeToGregorianLikeLongYear(dateTime) : formatDateTimeToGregorianLikeLongYearMonth(dateTime);
    });

    const availableBooks = computed<{ id: string, name: string, included: boolean }[]>(() => bookData.value.map(book => ({
        id: book.bookId,
        name: book.bookName,
        included: !excludedBookIds.value[book.bookId]
    })));

    const report = computed<HouseholdReport>(() => buildHouseholdReport(bookData.value.filter(book => !excludedBookIds.value[book.bookId]), (amount, currency) => {
        if (currency === defaultCurrency.value) {
            return amount;
        }

        const exchangedAmount = exchangeRatesStore.getExchangedAmount(parseBigDecimal(amount), currency, defaultCurrency.value);
        return exchangedAmount ? exchangedAmount.truncate().toDoubleNumber() : null;
    }));

    function loadBook(book: BookInfoResponse): Promise<HouseholdBookData> {
        const options = { bookId: book.isPersonal ? null : book.id };

        return Promise.all([
            handleApiResponse(services.getAllAccounts({ visibleOnly: false }, options), 'failed to load accounts for household report', 'Unable to retrieve account list'),
            handleApiResponse(services.getAllTransactionCategories(options), 'failed to load categories for household report', 'Unable to retrieve category list'),
            handleApiResponse(services.getTransactionStatistics({
                startTime: periodStartTime.value,
                endTime: periodEndTime.value,
                tagFilter: '',
                keyword: '',
                matchMode: KeywordMatchMode.Default.type,
                useTransactionTimezone: false
            }, options), 'failed to load statistics for household report', 'Unable to retrieve transaction statistics')
        ]).then(([accounts, categories, statistics]) => ({
            bookId: book.id,
            bookName: getBookDisplayName(book),
            accounts: accounts,
            categories: categories,
            statisticItems: statistics.items ?? []
        }));
    }

    function loadReport(): Promise<void> {
        loading.value = true;

        return booksStore.loadAllBooks().then(books => Promise.all(books.map(book => loadBook(book)))).then(data => {
            bookData.value = data;
            loading.value = false;
        }).catch(error => {
            loading.value = false;
            throw error;
        });
    }

    function setBookIncluded(bookId: string, included: boolean): void {
        if (!included && bookData.value.filter(book => !excludedBookIds.value[book.bookId]).length <= 1) {
            // keep at least one book in the report
            return;
        }

        excludedBookIds.value = { ...excludedBookIds.value, [bookId]: !included };
    }

    function setPeriodType(type: HouseholdReportPeriodType): Promise<void> {
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

    function formatAmount(amount: number): string {
        return formatAmountToLocalizedNumeralsWithCurrency(parseBigDecimal(amount), defaultCurrency.value);
    }

    function formatSavingsRate(rate: number | null): string {
        return rate === null ? tt('None') : formatPercentToLocalizedNumerals(rate, 1, '<0.1');
    }

    return {
        // states
        loading,
        periodType,
        // computed states
        periodDisplayName,
        availableBooks,
        report,
        // functions
        loadReport,
        setBookIncluded,
        setPeriodType,
        changePeriod,
        formatAmount,
        formatSavingsRate
    };
}
