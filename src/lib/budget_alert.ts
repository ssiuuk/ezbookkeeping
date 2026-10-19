import type { BudgetAlertInfoResponse } from '@/models/budget_alert.ts';

export const BUDGET_ALERT_EXCEEDED_THRESHOLD = 100;

export interface BudgetAlertNoticeItem {
    readonly alert: BudgetAlertInfoResponse;
    // ids of all undismissed alerts of the same category and period, they are dismissed together
    readonly alertIds: string[];
    readonly exceeded: boolean;
}

/**
 * Builds the notices of budget alerts, the dismissed alerts are left out and only the highest threshold
 * of each category in each period is shown. The latest notices come first.
 */
export function getBudgetAlertNoticeItems(alerts: BudgetAlertInfoResponse[], maxCount?: number): BudgetAlertNoticeItem[] {
    const itemMap: Record<string, { alert: BudgetAlertInfoResponse, alertIds: string[] }> = {};

    for (const alert of alerts) {
        if (alert.dismissed) {
            continue;
        }

        const key = `${alert.categoryId}_${alert.period}`;
        const item = itemMap[key];

        if (!item) {
            itemMap[key] = { alert: alert, alertIds: [alert.id] };
            continue;
        }

        item.alertIds.push(alert.id);

        if (alert.threshold > item.alert.threshold) {
            item.alert = alert;
        }
    }

    const items: BudgetAlertNoticeItem[] = Object.values(itemMap).map(item => ({
        alert: item.alert,
        alertIds: item.alertIds,
        exceeded: item.alert.threshold >= BUDGET_ALERT_EXCEEDED_THRESHOLD
    }));

    items.sort((item1, item2) => {
        if (item1.alert.time !== item2.alert.time) {
            return item2.alert.time - item1.alert.time;
        }

        return item2.alert.threshold - item1.alert.threshold;
    });

    if (maxCount && maxCount > 0 && items.length > maxCount) {
        return items.slice(0, maxCount);
    }

    return items;
}

/**
 * Returns the ids of all alerts of the given notices.
 */
export function getBudgetAlertNoticeItemIds(items: BudgetAlertNoticeItem[]): string[] {
    const ids: string[] = [];

    for (const item of items) {
        ids.push(...item.alertIds);
    }

    return ids;
}
