import { reversed } from '@/core/base.ts';
import type { DateTime, TextualYearMonthDay } from '@/core/datetime.ts';
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

export interface RepeatScheduledDates {
    readonly startDate: TextualYearMonthDay;
    readonly skipDate: TextualYearMonthDay;
}

// returns the dates of a scheduled template which repeats a transaction
// the start date is the transaction date, which the every N types use as the recurrence anchor (so a month end or leap day anchor is not lost by clamping),
// and the skip date tells the scheduled job that the occurrence on the transaction date has already been created, so it is not duplicated
export function getRepeatScheduledDates(transactionDateTime: DateTime): RepeatScheduledDates {
    const transactionDate = transactionDateTime.getGregorianCalendarYearDashMonthDashDay();

    return {
        startDate: transactionDate,
        skipDate: transactionDate
    };
}
