import { describe, expect, test } from 'vitest';

import type { DateTime } from '@/core/datetime.ts';
import { parseDateTimeFromUnixTimeWithTimezoneOffset } from '@/lib/datetime.ts';
import { getScheduledFrequencyValues, getRepeatScheduledDates } from '@/lib/template.ts';

const UTC_OFFSET = 480;

function localDateTime(year: number, month: number, day: number, hour: number = 12): DateTime {
    return parseDateTimeFromUnixTimeWithTimezoneOffset(Date.UTC(year, month - 1, day, hour, 0, 0) / 1000 - UTC_OFFSET * 60, UTC_OFFSET);
}

describe('getScheduledFrequencyValues', () => {
    test('parses comma separated values', () => {
        expect(getScheduledFrequencyValues('3,1,2')).toEqual([3, 1, 2]);
        expect(getScheduledFrequencyValues('')).toEqual([]);
        expect(getScheduledFrequencyValues(undefined)).toEqual([]);
    });
});

describe('getRepeatScheduledDates', () => {
    test('keeps the transaction date as the anchor and skips the already created occurrence', () => {
        expect(getRepeatScheduledDates(localDateTime(2026, 10, 18, 9))).toEqual({ startDate: '2026-10-18', skipDate: '2026-10-18' });
        expect(getRepeatScheduledDates(localDateTime(2026, 1, 31))).toEqual({ startDate: '2026-01-31', skipDate: '2026-01-31' });
    });

    test('does not move a future month end or leap day anchor to the clamped next occurrence', () => {
        expect(getRepeatScheduledDates(localDateTime(2027, 1, 31))).toEqual({ startDate: '2027-01-31', skipDate: '2027-01-31' });
        expect(getRepeatScheduledDates(localDateTime(2026, 11, 30))).toEqual({ startDate: '2026-11-30', skipDate: '2026-11-30' });
        expect(getRepeatScheduledDates(localDateTime(2028, 2, 29))).toEqual({ startDate: '2028-02-29', skipDate: '2028-02-29' });
    });

    test('uses the local date in the transaction timezone', () => {
        // 23:00 in UTC+8 is still 15:00 in UTC, and 00:00 in UTC+8 is still the previous day in UTC
        expect(getRepeatScheduledDates(localDateTime(2026, 10, 18, 23))).toEqual({ startDate: '2026-10-18', skipDate: '2026-10-18' });
        expect(getRepeatScheduledDates(localDateTime(2026, 10, 19, 0))).toEqual({ startDate: '2026-10-19', skipDate: '2026-10-19' });
    });
});
