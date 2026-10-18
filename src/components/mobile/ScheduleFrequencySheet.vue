<template>
    <f7-sheet swipe-to-close swipe-handler=".swipe-handler"
              style="height: auto" :opened="show" @sheet:open="onSheetOpen" @sheet:closed="onSheetClosed">
        <f7-toolbar class="toolbar-with-swipe-handler">
            <div class="swipe-handler"></div>
            <div class="left">
                <f7-link sheet-close icon-f7="xmark" :aria-label="tt('Close')"></f7-link>
            </div>
            <div class="right">
                <f7-button round fill icon-f7="checkmark_alt" :aria-label="tt('Apply')" @click="save"></f7-button>
            </div>
        </f7-toolbar>
        <f7-page-content class="margin-top">
            <div class="grid grid-cols-2 grid-gap">
                <div>
                    <div class="schedule-frequency-type-container">
                        <f7-list dividers class="schedule-frequency-type-list no-margin-vertical">
                            <f7-list-item link="#" no-chevron
                                          :key="item.option"
                                          :title="item.displayName"
                                          v-for="item in allFrequencyOptions"
                                          @click="changeFrequencyOption(item.option)">
                                <template #after>
                                    <f7-icon class="list-item-showing icon-with-direction" f7="chevron_right" v-if="currentOption === item.option"></f7-icon>
                                </template>
                            </f7-list-item>
                        </f7-list>
                    </div>
                </div>
                <div>
                    <div class="schedule-frequency-value-container">
                        <f7-list dividers class="schedule-frequency-value-list no-margin-vertical"
                                 v-if="currentOption === ScheduleFrequencyOption.Disabled">
                            <f7-list-item :title="tt('None')"></f7-list-item>
                        </f7-list>
                        <f7-list dividers class="schedule-frequency-value-list no-margin-vertical"
                                 v-if="isPresetOption">
                            <f7-list-item :title="currentDisplayFrequency"></f7-list-item>
                        </f7-list>
                        <f7-list dividers class="schedule-frequency-value-list no-margin-vertical"
                                 v-if="currentOption === ScheduleFrequencyOption.Custom">
                            <f7-list-item radio name="schedule-frequency-custom-unit"
                                          :key="unit.type"
                                          :value="unit.type"
                                          :checked="currentFrequencyType === unit.type"
                                          :title="unit.displayName"
                                          v-for="unit in allCustomIntervalUnits"
                                          @change="changeCustomIntervalUnit(unit.type)">
                            </f7-list-item>
                        </f7-list>
                        <f7-list dividers class="schedule-frequency-value-list no-margin-vertical"
                                 v-if="currentOption === ScheduleFrequencyOption.Custom">
                            <f7-list-item radio name="schedule-frequency-custom-interval"
                                          :class="isChecked(n.type) ? 'list-item-selected' : ''"
                                          :key="n.type"
                                          :value="n.type"
                                          :checked="isChecked(n.type)"
                                          :title="n.displayName"
                                          v-for="n in customIntervalItems"
                                          @change="setFrequencyValue">
                            </f7-list-item>
                        </f7-list>
                        <f7-list dividers class="schedule-frequency-value-list no-margin-vertical"
                                 v-if="currentOption === ScheduleFrequencyOption.DaysOfWeek">
                            <f7-list-item checkbox
                                          :class="isChecked(weekDay.type) ? 'list-item-selected' : ''"
                                          :key="weekDay.type"
                                          :value="weekDay.type"
                                          :checked="isChecked(weekDay.type)"
                                          :title="weekDay.displayName"
                                          v-for="weekDay in allWeekDays"
                                          @change="changeFrequencyValue">
                            </f7-list-item>
                        </f7-list>
                        <f7-list dividers class="schedule-frequency-value-list no-margin-vertical"
                                 v-if="currentOption === ScheduleFrequencyOption.DaysOfMonth">
                            <f7-list-item checkbox
                                          :class="isChecked(monthDay.type) ? 'list-item-selected' : ''"
                                          :key="monthDay.type"
                                          :value="monthDay.type"
                                          :checked="isChecked(monthDay.type)"
                                          :title="monthDay.displayName"
                                          v-for="monthDay in allAvailableMonthDays"
                                          @change="changeFrequencyValue">
                            </f7-list-item>
                        </f7-list>
                        <f7-list dividers class="schedule-frequency-value-list no-margin-vertical"
                                 v-if="currentOption === ScheduleFrequencyOption.DaysOfYear">
                            <f7-list-item checkbox
                                          :class="isChecked(monthAndDay.type) ? 'list-item-selected' : ''"
                                          :key="monthAndDay.type"
                                          :value="monthAndDay.type"
                                          :checked="isChecked(monthAndDay.type)"
                                          :title="monthAndDay.displayName"
                                          v-for="monthAndDay in allAvailableMonthAndDays"
                                          @change="changeFrequencyValue">
                            </f7-list-item>
                        </f7-list>
                    </div>
                </div>
            </div>
        </f7-page-content>
    </f7-sheet>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import {
    type CommonScheduleFrequencySelectionProps,
    type ScheduleFrequencyOptionItem,
    ScheduleFrequencyOption,
    getScheduleFrequencyOption,
    getScheduleFrequencyPreset,
    useScheduleFrequencySelectionBase
} from '@/components/base/ScheduleFrequencySelectionBase.ts';

import { useUserStore } from '@/stores/user.ts';

import { type TypeAndDisplayName, itemAndIndex } from '@/core/base.ts';
import { type WeekDayValue } from '@/core/datetime.ts';
import { ScheduledTemplateFrequencyType } from '@/core/template.ts';
import { sortNumbersArray } from '@/lib/common.ts';
import { scrollToSelectedItem } from '@/lib/ui/common.ts';
import { type Framework7Dom } from '@/lib/ui/mobile.ts';

interface MobileScheduleFrequencySelectionProps extends CommonScheduleFrequencySelectionProps {
    show: boolean;
}

const props = defineProps<MobileScheduleFrequencySelectionProps>();
const emit = defineEmits<{
    (e: 'update:type', value: number): void;
    (e: 'update:modelValue', value: string): void;
    (e: 'update:show', value: boolean): void;
}>();

const { tt } = useI18n();
const {
    allWeekDays,
    allAvailableMonthDays,
    allAvailableMonthAndDays,
    allCustomIntervalUnits,
    getAllFrequencyOptions,
    getCustomIntervalItems,
    getFrequencyDisplayText,
    getFrequencyValues
} = useScheduleFrequencySelectionBase();

const userStore = useUserStore();

const currentFrequencyType = ref<number>(props.type);
const currentFrequencyValue = ref<number[]>(getFrequencyValues(props.modelValue));
const currentOption = ref<ScheduleFrequencyOption>(getScheduleFrequencyOption(props.type, currentFrequencyValue.value));

const firstDayOfWeek = computed<WeekDayValue>(() => userStore.currentUserFirstDayOfWeek);

const allFrequencyOptions = computed<ScheduleFrequencyOptionItem[]>(() => getAllFrequencyOptions(props.disabledText));
const customIntervalItems = computed<TypeAndDisplayName[]>(() => getCustomIntervalItems(currentFrequencyType.value));
const isPresetOption = computed<boolean>(() => !!getScheduleFrequencyPreset(currentOption.value));
const currentDisplayFrequency = computed<string>(() => getFrequencyDisplayText(currentFrequencyType.value, currentFrequencyValue.value.join(','), props.disabledText));

function isChecked(value: number): boolean {
    return currentFrequencyValue.value.indexOf(value) >= 0;
}

function changeFrequencyOption(option: ScheduleFrequencyOption): void {
    if (currentOption.value === option) {
        return;
    }

    const preset = getScheduleFrequencyPreset(option);
    currentOption.value = option;

    if (preset) {
        currentFrequencyType.value = preset.type;
        currentFrequencyValue.value = [preset.value];
    } else if (option === ScheduleFrequencyOption.Custom) {
        if (!ScheduledTemplateFrequencyType.isEveryNInterval(currentFrequencyType.value)) {
            currentFrequencyType.value = ScheduledTemplateFrequencyType.EveryNDays.type;
            currentFrequencyValue.value = [1];
        }
    } else if (option === ScheduleFrequencyOption.DaysOfWeek) {
        currentFrequencyType.value = ScheduledTemplateFrequencyType.Weekly.type;
        currentFrequencyValue.value = [firstDayOfWeek.value];
    } else if (option === ScheduleFrequencyOption.DaysOfMonth) {
        currentFrequencyType.value = ScheduledTemplateFrequencyType.Monthly.type;
        currentFrequencyValue.value = [1];
    } else if (option === ScheduleFrequencyOption.DaysOfYear) {
        currentFrequencyType.value = ScheduledTemplateFrequencyType.Yearly.type;
        currentFrequencyValue.value = [101];
    } else {
        currentFrequencyType.value = ScheduledTemplateFrequencyType.Disabled.type;
        currentFrequencyValue.value = [];
    }
}

function changeCustomIntervalUnit(type: number): void {
    currentFrequencyType.value = type;
    currentFrequencyValue.value = [currentFrequencyValue.value[0] || 1];
}

function setFrequencyValue(e: Event): void {
    const currentValue = parseInt((e.target as HTMLInputElement).value);

    currentFrequencyValue.value.splice(0, currentFrequencyValue.value.length, currentValue);
}

function changeFrequencyValue(e: Event): void {
    const currentValue = parseInt((e.target as HTMLInputElement).value);

    if ((e.target as HTMLInputElement).checked) {
        for (const value of currentFrequencyValue.value) {
            if (value === currentValue) {
                return;
            }
        }

        currentFrequencyValue.value.push(currentValue);
    } else {
        for (const [value, index] of itemAndIndex(currentFrequencyValue.value)) {
            if (value === currentValue) {
                currentFrequencyValue.value.splice(index, 1);
                break;
            }
        }
    }
}

function save(): void {
    emit('update:type', currentFrequencyType.value);
    emit('update:modelValue', sortNumbersArray(currentFrequencyValue.value).join(','));
    emit('update:show', false);
}

function close(): void {
    emit('update:show', false);
}

function onSheetOpen(event: { $el: Framework7Dom }): void {
    currentFrequencyType.value = props.type;
    currentFrequencyValue.value = getFrequencyValues(props.modelValue);
    currentOption.value = getScheduleFrequencyOption(currentFrequencyType.value, currentFrequencyValue.value);
    scrollToSelectedItem(event.$el[0], '.schedule-frequency-value-container', '.schedule-frequency-value-container', 'li.list-item-selected');
}

function onSheetClosed(): void {
    close();
}
</script>

<style>
.schedule-frequency-type-container, .schedule-frequency-value-container {
    height: 260px;
    overflow-y: auto;
}

.schedule-frequency-type-list.list .item-inner {
    padding-inline-end: 6px;
}

.schedule-frequency-value-list-list.list .item-content {
    padding-inline-start: 0;
}
</style>
