import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useProjectDateRangeText } from './ProjectPageBase.ts';

import { useUserStore } from '@/stores/user.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useExchangeRatesStore } from '@/stores/exchangeRates.ts';
import { useProjectsStore } from '@/stores/project.ts';

import { KeywordMatchMode } from '@/core/text.ts';
import { TransactionType } from '@/core/transaction.ts';
import { DISPLAY_HIDDEN_AMOUNT } from '@/consts/numeral.ts';

import type { ProjectInfoResponse } from '@/models/project.ts';
import type { TransactionCategory } from '@/models/transaction_category.ts';
import type { TransactionInfoResponse, TransactionStatisticResponseItem } from '@/models/transaction.ts';

import { type ProjectReport, buildProjectReport } from '@/lib/project_report.ts';
import { parseDateTimeFromUnixTime } from '@/lib/datetime.ts';
import { parseBigDecimal } from '@/lib/numeral.ts';
import { handleApiResponse } from '@/lib/api.ts';
import services from '@/lib/services.ts';

const TRANSACTION_PAGE_COUNT = 50;

export function useProjectReportPageBase() {
    const { tt, formatDateTimeToShortDate, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();
    const { getProjectDateRangeText } = useProjectDateRangeText();

    const userStore = useUserStore();
    const accountsStore = useAccountsStore();
    const transactionCategoriesStore = useTransactionCategoriesStore();
    const exchangeRatesStore = useExchangeRatesStore();
    const projectsStore = useProjectsStore();

    const projectId = ref<string>('');
    const loading = ref<boolean>(true);
    const loadingMore = ref<boolean>(false);
    const project = ref<ProjectInfoResponse | null>(null);
    const statisticItems = ref<TransactionStatisticResponseItem[]>([]);
    const transactions = ref<TransactionInfoResponse[]>([]);
    const nextTimeSequenceId = ref<number | undefined>(undefined);

    const defaultCurrency = computed<string>(() => userStore.currentUserDefaultCurrency);
    const hasMoreTransactions = computed<boolean>(() => !!nextTimeSequenceId.value);
    const dateRangeText = computed<string>(() => getProjectDateRangeText(project.value));

    const report = computed<ProjectReport>(() => buildProjectReport(statisticItems.value, accountsStore.allAccountsMap, transactionCategoriesStore.allTransactionCategoriesMap, (amount, currency) => {
        if (currency === defaultCurrency.value) {
            return amount;
        }

        const exchangedAmount = exchangeRatesStore.getExchangedAmount(parseBigDecimal(amount), currency, defaultCurrency.value);
        return exchangedAmount ? exchangedAmount.truncate().toDoubleNumber() : null;
    }));

    function loadTransactions(maxTime: number): Promise<void> {
        return handleApiResponse(services.getTransactions({
            maxTime: maxTime,
            minTime: 0,
            count: TRANSACTION_PAGE_COUNT,
            page: 1,
            withCount: false,
            type: 0,
            categoryIds: '',
            accountIds: '',
            tagFilter: '',
            amountFilter: '',
            keyword: '',
            matchMode: KeywordMatchMode.Default.type,
            projectId: projectId.value
        }), 'failed to retrieve project transactions', 'Unable to retrieve transaction list').then(result => {
            if (maxTime > 0) {
                transactions.value.push(...result.items);
            } else {
                transactions.value = result.items;
            }

            nextTimeSequenceId.value = result.nextTimeSequenceId;
        });
    }

    function loadAll(id: string): Promise<void> {
        projectId.value = id;
        loading.value = true;

        return Promise.all([
            accountsStore.loadAllAccounts({ force: false }),
            transactionCategoriesStore.loadAllCategories({ force: false }),
            projectsStore.getProject({ projectId: id }),
            handleApiResponse(services.getTransactionStatistics({
                startTime: 0,
                endTime: 0,
                tagFilter: '',
                keyword: '',
                matchMode: KeywordMatchMode.Default.type,
                useTransactionTimezone: false,
                projectId: id
            }), 'failed to retrieve project statistics', 'Unable to retrieve transaction statistics'),
            loadTransactions(0)
        ]).then(([, , projectInfo, statistics]) => {
            project.value = projectInfo;
            statisticItems.value = statistics.items ?? [];
            loading.value = false;
        }).catch(error => {
            loading.value = false;
            throw error;
        });
    }

    function loadMoreTransactions(): Promise<void> {
        if (!nextTimeSequenceId.value || loadingMore.value) {
            return Promise.resolve();
        }

        loadingMore.value = true;

        return loadTransactions(nextTimeSequenceId.value).finally(() => {
            loadingMore.value = false;
        });
    }

    function formatAmount(amount: number): string {
        return formatAmountToLocalizedNumeralsWithCurrency(parseBigDecimal(amount), defaultCurrency.value);
    }

    function getTransactionCategory(transaction: TransactionInfoResponse): TransactionCategory | undefined {
        if (transaction.type === TransactionType.ModifyBalance) {
            return undefined;
        }

        return transactionCategoriesStore.allTransactionCategoriesMap[transaction.categoryId];
    }

    function getTransactionTitle(transaction: TransactionInfoResponse): string {
        const category = getTransactionCategory(transaction);

        if (category) {
            return category.name;
        } else if (transaction.type === TransactionType.Income) {
            return tt('Income');
        } else if (transaction.type === TransactionType.Expense) {
            return tt('Expense');
        } else if (transaction.type === TransactionType.Transfer) {
            return tt('Transfer');
        }

        return tt('Transaction');
    }

    function getTransactionDescription(transaction: TransactionInfoResponse): string {
        const accountName = accountsStore.allAccountsMap[transaction.sourceAccountId]?.name ?? '';
        const date = formatDateTimeToShortDate(parseDateTimeFromUnixTime(transaction.time));
        return `${date}${accountName ? ` · ${accountName}` : ''}${transaction.comment ? ` · ${transaction.comment}` : ''}`;
    }

    function getTransactionAmount(transaction: TransactionInfoResponse): string {
        const amount = transaction.hideAmount ? DISPLAY_HIDDEN_AMOUNT : parseBigDecimal(transaction.sourceAmount);
        const currency = accountsStore.allAccountsMap[transaction.sourceAccountId]?.currency ?? '';
        const formatted = formatAmountToLocalizedNumeralsWithCurrency(amount, currency);

        if (transaction.type === TransactionType.Expense) {
            return `-${formatted}`;
        } else if (transaction.type === TransactionType.Income) {
            return `+${formatted}`;
        }

        return formatted;
    }

    function getTransactionAmountClass(transaction: TransactionInfoResponse): string {
        if (transaction.type === TransactionType.Income) {
            return 'text-income';
        } else if (transaction.type === TransactionType.Expense) {
            return 'text-expense';
        }

        return '';
    }

    return {
        // states
        projectId,
        loading,
        loadingMore,
        project,
        transactions,
        // computed states
        defaultCurrency,
        hasMoreTransactions,
        dateRangeText,
        report,
        // functions
        loadAll,
        loadMoreTransactions,
        formatAmount,
        getTransactionCategory,
        getTransactionTitle,
        getTransactionDescription,
        getTransactionAmount,
        getTransactionAmountClass
    };
}
