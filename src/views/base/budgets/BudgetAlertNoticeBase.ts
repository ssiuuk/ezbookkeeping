import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import type { TextualYearMonth } from '@/core/datetime.ts';
import type { BudgetAlertInfoResponse } from '@/models/budget_alert.ts';

import {
    type BudgetAlertNoticeItem,
    getBudgetAlertNoticeItems,
    getBudgetAlertNoticeItemIds
} from '@/lib/budget_alert.ts';
import { getYearMonthFirstUnixTime, parseDateTimeFromUnixTime } from '@/lib/datetime.ts';
import { parseBigDecimal } from '@/lib/numeral.ts';
import { isBudgetAlertEnabled } from '@/lib/server_settings.ts';
import { handleApiResponse } from '@/lib/api.ts';
import services from '@/lib/services.ts';

const maxBudgetAlertNoticeCount = 5;
const loadBudgetAlertCount = 20;

export function useBudgetAlertNoticeBase() {
    const { tt, formatDateTimeToGregorianLikeLongYearMonth, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();

    const alerts = ref<BudgetAlertInfoResponse[]>([]);
    const dismissing = ref<boolean>(false);

    const noticeItems = computed<BudgetAlertNoticeItem[]>(() => getBudgetAlertNoticeItems(alerts.value, maxBudgetAlertNoticeCount));

    function getNoticeText(item: BudgetAlertNoticeItem): string {
        const alert = item.alert;
        const periodFirstUnixTime = getYearMonthFirstUnixTime(alert.period as TextualYearMonth);
        const params = {
            category: alert.categoryName,
            period: periodFirstUnixTime ? formatDateTimeToGregorianLikeLongYearMonth(parseDateTimeFromUnixTime(periodFirstUnixTime)) : alert.period,
            percent: alert.threshold,
            spent: formatAmountToLocalizedNumeralsWithCurrency(parseBigDecimal(alert.spent), alert.currency),
            budget: formatAmountToLocalizedNumeralsWithCurrency(parseBigDecimal(alert.budget), alert.currency)
        };

        if (item.exceeded) {
            return tt('{category}: the budget for {period} has been exceeded ({spent} of {budget})', params);
        } else {
            return tt('{category}: {percent}% of the budget for {period} has been spent ({spent} of {budget})', params);
        }
    }

    function loadAlerts(): Promise<void> {
        if (!isBudgetAlertEnabled()) {
            alerts.value = [];
            return Promise.resolve();
        }

        return handleApiResponse(services.getBudgetAlerts({ count: loadBudgetAlertCount }), 'failed to retrieve budget alerts', 'Unable to retrieve budget alerts').then(result => {
            alerts.value = result;
        });
    }

    function dismissAlerts(ids: string[]): Promise<void> {
        if (!ids.length) {
            return Promise.resolve();
        }

        dismissing.value = true;

        return handleApiResponse(services.dismissBudgetAlerts({ ids: ids }), 'failed to dismiss budget alerts', 'Unable to dismiss budget alerts').then(() => {
            alerts.value = alerts.value.filter(alert => !ids.includes(alert.id));
            dismissing.value = false;
        }).catch(error => {
            dismissing.value = false;
            throw error;
        });
    }

    function dismissNotice(item: BudgetAlertNoticeItem): Promise<void> {
        return dismissAlerts(item.alertIds);
    }

    function dismissAllNotices(): Promise<void> {
        return dismissAlerts(getBudgetAlertNoticeItemIds(getBudgetAlertNoticeItems(alerts.value)));
    }

    return {
        // states
        dismissing,
        // computed states
        noticeItems,
        // functions
        getNoticeText,
        loadAlerts,
        dismissNotice,
        dismissAllNotices
    };
}
