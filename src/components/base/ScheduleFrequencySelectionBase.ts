import { computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useUserStore } from '@/stores/user.ts';

import { type TypeAndDisplayName, itemAndIndex } from '@/core/base.ts';
import { type DateTime } from '@/core/datetime.ts';
import { ScheduledTemplateFrequencyType } from '@/core/template.ts';
import { SCHEDULED_TRANSACTION_MAX_FREQUENCY_INTERVAL } from '@/consts/transaction.ts';

import { sortNumbersArray } from '@/lib/common.ts';
import { getCurrentDateTime } from '@/lib/datetime.ts';
import { getScheduledFrequencyValues } from '@/lib/template.ts';

export interface CommonScheduleFrequencySelectionProps {
    type: number;
    modelValue: string;
    disabled?: boolean;
    readonly?: boolean;
    label?: string;
    disabledText?: string;
}

export enum ScheduleFrequencyOption {
    Disabled = 'disabled',
    Daily = 'daily',
    Weekly = 'weekly',
    Fortnightly = 'fortnightly',
    Monthly = 'monthly',
    EveryThreeMonths = 'everyThreeMonths',
    Yearly = 'yearly',
    Custom = 'custom',
    DaysOfWeek = 'daysOfWeek',
    DaysOfMonth = 'daysOfMonth',
    DaysOfYear = 'daysOfYear'
}

export interface ScheduleFrequencyOptionItem {
    readonly option: ScheduleFrequencyOption;
    readonly displayName: string;
}

interface ScheduleFrequencyPreset {
    readonly type: number;
    readonly value: number;
}

const SCHEDULE_FREQUENCY_PRESETS: Partial<Record<ScheduleFrequencyOption, ScheduleFrequencyPreset>> = {
    [ScheduleFrequencyOption.Daily]: { type: ScheduledTemplateFrequencyType.Daily.type, value: 0 },
    [ScheduleFrequencyOption.Weekly]: { type: ScheduledTemplateFrequencyType.EveryNWeeks.type, value: 1 },
    [ScheduleFrequencyOption.Fortnightly]: { type: ScheduledTemplateFrequencyType.EveryNWeeks.type, value: 2 },
    [ScheduleFrequencyOption.Monthly]: { type: ScheduledTemplateFrequencyType.EveryNMonths.type, value: 1 },
    [ScheduleFrequencyOption.EveryThreeMonths]: { type: ScheduledTemplateFrequencyType.EveryNMonths.type, value: 3 },
    [ScheduleFrequencyOption.Yearly]: { type: ScheduledTemplateFrequencyType.EveryNYears.type, value: 1 }
};

export function getScheduleFrequencyPreset(option: ScheduleFrequencyOption): ScheduleFrequencyPreset | undefined {
    return SCHEDULE_FREQUENCY_PRESETS[option];
}

export function getScheduleFrequencyOption(type: number, values: number[]): ScheduleFrequencyOption {
    if (type === ScheduledTemplateFrequencyType.Disabled.type) {
        return ScheduleFrequencyOption.Disabled;
    } else if (type === ScheduledTemplateFrequencyType.Daily.type) {
        return ScheduleFrequencyOption.Daily;
    } else if (type === ScheduledTemplateFrequencyType.Weekly.type) {
        return ScheduleFrequencyOption.DaysOfWeek;
    } else if (type === ScheduledTemplateFrequencyType.Monthly.type) {
        return ScheduleFrequencyOption.DaysOfMonth;
    } else if (type === ScheduledTemplateFrequencyType.Yearly.type) {
        return ScheduleFrequencyOption.DaysOfYear;
    }

    for (const [option, preset] of Object.entries(SCHEDULE_FREQUENCY_PRESETS)) {
        if (preset && preset.type === type && values.length === 1 && values[0] === preset.value) {
            return option as ScheduleFrequencyOption;
        }
    }

    return ScheduleFrequencyOption.Custom;
}

export function useScheduleFrequencySelectionBase() {
    const {
        tt,
        getAllWeekDays,
        getAvailableMonthDays,
        getMultiMonthAndDayLongNames,
        getMultiMonthdayShortNames,
        getMultiWeekdayLongNames,
        formatDateTimeToLongMonthDay,
        formatNumberToLocalizedNumerals
    } = useI18n();
    const userStore = useUserStore();

    const allWeekDays = computed<TypeAndDisplayName[]>(() => getAllWeekDays(userStore.currentUserFirstDayOfWeek));

    const allAvailableMonthDays = computed<TypeAndDisplayName[]>(() => getAvailableMonthDays(28, 3));
    const allAvailableMonthAndDays = computed<TypeAndDisplayName[]>(() => {
        const ret: TypeAndDisplayName[] = [];
        const now: DateTime = getCurrentDateTime();
        const maxDaysOfMonth: number[] = [31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];

        for (const [days, index] of itemAndIndex(maxDaysOfMonth)) {
            const month = index + 1;

            for (let day = 1; day <= days; day++) {
                const dateTime = now.set({
                    month: month,
                    dayOfMonth: day,
                    hour: 0,
                    minute: 0,
                    second: 0,
                    millisecond: 0
                });

                ret.push({
                    type: month * 100 + day,
                    displayName: formatDateTimeToLongMonthDay(dateTime)
                });
            }
        }

        return ret;
    });

    const allCustomIntervalUnits = computed<TypeAndDisplayName[]>(() => [
        { type: ScheduledTemplateFrequencyType.EveryNDays.type, displayName: tt('Days') },
        { type: ScheduledTemplateFrequencyType.EveryNWeeks.type, displayName: tt('Weeks') },
        { type: ScheduledTemplateFrequencyType.EveryNMonths.type, displayName: tt('Months') },
        { type: ScheduledTemplateFrequencyType.EveryNYears.type, displayName: tt('Years') }
    ]);

    function getAllFrequencyOptions(disabledText?: string): ScheduleFrequencyOptionItem[] {
        return [
            { option: ScheduleFrequencyOption.Disabled, displayName: disabledText || tt('Disabled') },
            { option: ScheduleFrequencyOption.Daily, displayName: tt('Daily') },
            { option: ScheduleFrequencyOption.Weekly, displayName: tt('Weekly') },
            { option: ScheduleFrequencyOption.Fortnightly, displayName: tt('Fortnightly') },
            { option: ScheduleFrequencyOption.Monthly, displayName: tt('Monthly') },
            { option: ScheduleFrequencyOption.EveryThreeMonths, displayName: tt('Every 3 Months') },
            { option: ScheduleFrequencyOption.Yearly, displayName: tt('Yearly') },
            { option: ScheduleFrequencyOption.Custom, displayName: tt('Custom') },
            { option: ScheduleFrequencyOption.DaysOfWeek, displayName: tt('Days of Week') },
            { option: ScheduleFrequencyOption.DaysOfMonth, displayName: tt('Days of Month') },
            { option: ScheduleFrequencyOption.DaysOfYear, displayName: tt('Days of Year') }
        ];
    }

    function getEveryNIntervalDisplayText(type: number, n: number): string {
        if (type === ScheduledTemplateFrequencyType.EveryNDays.type) {
            return tt('format.misc.everyNDays', { n: formatNumberToLocalizedNumerals(n) });
        } else if (type === ScheduledTemplateFrequencyType.EveryNWeeks.type) {
            return n === 1 ? tt('Weekly') : tt('Every {n} weeks', { n: formatNumberToLocalizedNumerals(n) });
        } else if (type === ScheduledTemplateFrequencyType.EveryNMonths.type) {
            return n === 1 ? tt('Monthly') : tt('Every {n} months', { n: formatNumberToLocalizedNumerals(n) });
        } else if (type === ScheduledTemplateFrequencyType.EveryNYears.type) {
            return n === 1 ? tt('Yearly') : tt('Every {n} years', { n: formatNumberToLocalizedNumerals(n) });
        } else {
            return '';
        }
    }

    function getCustomIntervalItems(type: number): TypeAndDisplayName[] {
        const ret: TypeAndDisplayName[] = [];

        for (let n = 1; n <= SCHEDULED_TRANSACTION_MAX_FREQUENCY_INTERVAL; n++) {
            ret.push({
                type: n,
                displayName: getEveryNIntervalDisplayText(type, n)
            });
        }

        return ret;
    }

    function getFrequencyDisplayText(type: number, value: string, disabledText?: string): string {
        const values = getFrequencyValues(value);

        if (type === ScheduledTemplateFrequencyType.Disabled.type) {
            return disabledText || tt('Disabled');
        } else if (type === ScheduledTemplateFrequencyType.Daily.type) {
            return tt('Daily');
        } else if (ScheduledTemplateFrequencyType.isEveryNInterval(type)) {
            if (values.length && values[0]) {
                return getEveryNIntervalDisplayText(type, values[0]);
            } else {
                return tt(ScheduledTemplateFrequencyType.valueOf(type)?.name ?? 'Custom');
            }
        } else if (type === ScheduledTemplateFrequencyType.Weekly.type) {
            if (values.length) {
                return tt('format.misc.everyMultiDaysOfWeek', {
                    days: getMultiWeekdayLongNames(values, userStore.currentUserFirstDayOfWeek)
                });
            } else {
                return tt('Weekly');
            }
        } else if (type === ScheduledTemplateFrequencyType.Monthly.type) {
            if (values.length) {
                return tt('format.misc.everyMultiDaysOfMonth', {
                    days: getMultiMonthdayShortNames(values)
                });
            } else {
                return tt('Monthly');
            }
        } else if (type === ScheduledTemplateFrequencyType.Yearly.type) {
            if (values.length) {
                return tt('format.misc.everyMultiDaysOfYear', {
                    days: getMultiMonthAndDayLongNames(values)
                });
            } else {
                return tt('Yearly');
            }
        } else {
            return '';
        }
    }

    function getFrequencyValues(value: string): number[] {
        return sortNumbersArray(getScheduledFrequencyValues(value));
    }

    return {
        // computed states
        allWeekDays,
        allAvailableMonthDays,
        allAvailableMonthAndDays,
        allCustomIntervalUnits,
        // functions
        getAllFrequencyOptions,
        getCustomIntervalItems,
        getFrequencyDisplayText,
        getFrequencyValues
    };
}
