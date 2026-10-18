import { describe, expect, test } from 'vitest';

import type { DateTime } from '@/core/datetime.ts';
import { ScheduledTemplateFrequencyType } from '@/core/template.ts';
import { parseDateTimeFromUnixTimeWithTimezoneOffset } from '@/lib/datetime.ts';
import { getScheduledFrequencyValues, getRepeatScheduledStartDate } from '@/lib/template.ts';

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

describe('getRepeatScheduledStartDate', () => {
    const today = localDateTime(2026, 10, 18);

    test('keeps the transaction date as the anchor for every N types when it is not in the future', () => {
        expect(getRepeatScheduledStartDate(ScheduledTemplateFrequencyType.EveryNWeeks.type, '2', localDateTime(2026, 10, 18, 9), today)).toBe('2026-10-18');
        expect(getRepeatScheduledStartDate(ScheduledTemplateFrequencyType.EveryNMonths.type, '1', localDateTime(2026, 1, 31), today)).toBe('2026-01-31');
        expect(getRepeatScheduledStartDate(ScheduledTemplateFrequencyType.EveryNDays.type, '3', localDateTime(2026, 10, 1), today)).toBe('2026-10-01');
    });

    test('uses the next occurrence for every N types when the transaction date is in the future', () => {
        expect(getRepeatScheduledStartDate(ScheduledTemplateFrequencyType.EveryNDays.type, '3', localDateTime(2026, 10, 20), today)).toBe('2026-10-23');
        expect(getRepeatScheduledStartDate(ScheduledTemplateFrequencyType.EveryNWeeks.type, '2', localDateTime(2026, 10, 20), today)).toBe('2026-11-03');
        expect(getRepeatScheduledStartDate(ScheduledTemplateFrequencyType.EveryNMonths.type, '1', localDateTime(2027, 1, 31), today)).toBe('2027-02-28');
        expect(getRepeatScheduledStartDate(ScheduledTemplateFrequencyType.EveryNMonths.type, '3', localDateTime(2026, 11, 30), today)).toBe('2027-02-28');
        expect(getRepeatScheduledStartDate(ScheduledTemplateFrequencyType.EveryNYears.type, '1', localDateTime(2028, 2, 29), today)).toBe('2029-02-28');
    });

    test('starts from the next day for the other frequency types', () => {
        expect(getRepeatScheduledStartDate(ScheduledTemplateFrequencyType.Daily.type, '0', localDateTime(2026, 10, 18), today)).toBe('2026-10-19');
        expect(getRepeatScheduledStartDate(ScheduledTemplateFrequencyType.Weekly.type, '1,3', localDateTime(2026, 12, 31), today)).toBe('2027-01-01');
        expect(getRepeatScheduledStartDate(ScheduledTemplateFrequencyType.Monthly.type, '15', localDateTime(2026, 9, 1), today)).toBe('2026-09-02');
    });

    test('uses the local date in the transaction timezone', () => {
        // 23:00 in UTC+8 is still 15:00 in UTC, and 00:00 in UTC+8 is still the previous day in UTC
        expect(getRepeatScheduledStartDate(ScheduledTemplateFrequencyType.EveryNWeeks.type, '1', localDateTime(2026, 10, 18, 23), today)).toBe('2026-10-18');
        expect(getRepeatScheduledStartDate(ScheduledTemplateFrequencyType.EveryNWeeks.type, '1', localDateTime(2026, 10, 19, 0), today)).toBe('2026-10-26');
    });
});
