import { reversed } from '@/core/base.ts';
import type { DateTime, DateTimeUnit, TextualYearMonthDay } from '@/core/datetime.ts';
import { ScheduledTemplateFrequencyType } from '@/core/template.ts';
import { TransactionTemplate } from '@/models/transaction_template.ts';

export function isNoAvailableTemplate(templates: TransactionTemplate[], showHidden: boolean): boolean {
    for (const template of templates) {
        if (showHidden || !template.hidden) {
            return false;
        }
    }

    return true;
}

export function getAvailableTemplateCount(templates: TransactionTemplate[], showHidden: boolean): number {
    let count = 0;

    for (const template of templates) {
        if (showHidden || !template.hidden) {
            count++;
        }
    }

    return count;
}

export function getFirstShowingId(templates: TransactionTemplate[], showHidden: boolean): string | null {
    for (const template of templates) {
        if (showHidden || !template.hidden) {
            return template.id;
        }
    }

    return null;
}

export function getLastShowingId(templates: TransactionTemplate[], showHidden: boolean): string | null {
    for (const template of reversed(templates)) {
        if (showHidden || !template.hidden) {
            return template.id;
        }
    }

    return null;
}

export function getScheduledFrequencyValues(frequency: string | undefined): number[] {
    const ret: number[] = [];

    for (const item of (frequency || '').split(',')) {
        if (item) {
            ret.push(parseInt(item));
        }
    }

    return ret;
}

function getEveryNIntervalAmountAndUnit(frequencyType: number, n: number): { amount: number, unit: DateTimeUnit } | null {
    if (frequencyType === ScheduledTemplateFrequencyType.EveryNDays.type) {
        return { amount: n, unit: 'days' };
    } else if (frequencyType === ScheduledTemplateFrequencyType.EveryNWeeks.type) {
        return { amount: n * 7, unit: 'days' };
    } else if (frequencyType === ScheduledTemplateFrequencyType.EveryNMonths.type) {
        return { amount: n, unit: 'months' };
    } else if (frequencyType === ScheduledTemplateFrequencyType.EveryNYears.type) {
        return { amount: n, unit: 'years' };
    } else {
        return null;
    }
}

// returns the start date of a scheduled template which repeats a transaction, so that its first occurrence is the next one after the transaction date
// the every N types use the start date as the anchor, so the transaction date is kept unless the scheduled job could still create a transaction on that date
export function getRepeatScheduledStartDate(frequencyType: number, frequency: string, transactionDateTime: DateTime, currentDateTime: DateTime): TextualYearMonthDay {
    const transactionDate = transactionDateTime.getGregorianCalendarYearDashMonthDashDay();

    if (!ScheduledTemplateFrequencyType.isEveryNInterval(frequencyType)) {
        return transactionDateTime.add(1, 'days').getGregorianCalendarYearDashMonthDashDay();
    }

    // the scheduled job runs at the start of each day, so it has already run for today and earlier dates
    if (transactionDate <= currentDateTime.getGregorianCalendarYearDashMonthDashDay()) {
        return transactionDate;
    }

    const values = getScheduledFrequencyValues(frequency);
    const interval = getEveryNIntervalAmountAndUnit(frequencyType, values.length === 1 && values[0] ? values[0] : 1);

    if (!interval) {
        return transactionDate;
    }

    return transactionDateTime.add(interval.amount, interval.unit).getGregorianCalendarYearDashMonthDashDay();
}
