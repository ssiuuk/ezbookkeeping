import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useUserStore } from '@/stores/user.ts';
import { useBooksStore } from '@/stores/book.ts';

import type { DataStatisticsResponse, DisplayDataStatistics } from '@/models/data_management.ts';

import { parseBigDecimal } from '@/lib/numeral.ts';

export function useExportFileName() {
    const { tt } = useI18n();

    const userStore = useUserStore();
    const booksStore = useBooksStore();

    function getExportFileName(fileExtension: string): string {
        const nickname = userStore.currentUserNickname;
        // in a shared book, the exported data is the shared book's data, so the book name is also put into the file name
        const bookName = booksStore.isInSharedBook ? booksStore.currentBook?.name : undefined;
        const ownerName = bookName ? (nickname ? `${bookName}_${nickname}` : bookName) : nickname;

        if (ownerName) {
            return tt('dataExport.exportFilename', {
                nickname: ownerName
            }) + '.' + fileExtension;
        }

        return tt('dataExport.defaultExportFilename') + '.' + fileExtension;
    }

    return {
        getExportFileName
    };
}

export function useDataManagementPageBase() {
    const { formatBigDecimalToLocalizedNumerals } = useI18n();
    const { getExportFileName } = useExportFileName();

    const booksStore = useBooksStore();

    const dataStatistics = ref<DataStatisticsResponse | null>(null);

    // clearing data is destructive, so it is only available in the user's own book
    const isClearingDataAvailable = computed<boolean>(() => !booksStore.isInSharedBook);

    const displayDataStatistics = computed<DisplayDataStatistics | null>(() => {
        if (!dataStatistics.value) {
            return null;
        }

        return {
            totalTransactionCount: formatBigDecimalToLocalizedNumerals(parseBigDecimal(dataStatistics.value.totalTransactionCount)),
            totalAccountCount: formatBigDecimalToLocalizedNumerals(parseBigDecimal(dataStatistics.value.totalAccountCount)),
            totalTransactionCategoryCount: formatBigDecimalToLocalizedNumerals(parseBigDecimal(dataStatistics.value.totalTransactionCategoryCount)),
            totalTransactionTagCount: formatBigDecimalToLocalizedNumerals(parseBigDecimal(dataStatistics.value.totalTransactionTagCount)),
            totalTransactionPictureCount: formatBigDecimalToLocalizedNumerals(parseBigDecimal(dataStatistics.value.totalTransactionPictureCount)),
            totalExplorationCount: formatBigDecimalToLocalizedNumerals(parseBigDecimal(dataStatistics.value.totalExplorationCount)),
            totalTransactionTemplateCount: formatBigDecimalToLocalizedNumerals(parseBigDecimal(dataStatistics.value.totalTransactionTemplateCount)),
            totalScheduledTransactionCount: formatBigDecimalToLocalizedNumerals(parseBigDecimal(dataStatistics.value.totalScheduledTransactionCount)),
            totalCustomIconCount: formatBigDecimalToLocalizedNumerals(parseBigDecimal(dataStatistics.value.totalCustomIconCount))
        };
    });

    return {
        // states
        dataStatistics,
        // computed states
        isClearingDataAvailable,
        displayDataStatistics,
        // functions
        getExportFileName
    }
}
