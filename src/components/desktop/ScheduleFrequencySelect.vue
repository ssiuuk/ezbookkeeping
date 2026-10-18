<template>
    <v-select
        ref="frequencySelect"
        persistent-placeholder
        :readonly="readonly"
        :disabled="disabled"
        :label="label"
        :menu-props="{ contentClass: 'schedule-frequency-select-menu' }"
        :model-value="frequencyType"
        v-model:menu="menuState"
        @update:menu="onMenuStateChanged"
    >
        <template #selection>
            <span class="text-truncate cursor-pointer">{{ displayFrequency }}</span>
        </template>

        <template #no-data>
            <div ref="dropdownMenu" class="schedule-frequency-container">
                <div class="schedule-frequency-type-container">
                    <v-list ref="frequencyTypeList" @keydown.capture="onListKeydown($event, 'left')">
                        <v-list-item :class="{ 'v-list-item--active text-primary': item.option === currentOption }"
                                     :key="item.option" :title="item.displayName"
                                     v-for="item in allFrequencyOptions"
                                     @click="selectOption(item.option)">
                        </v-list-item>
                    </v-list>
                </div>
                <div class="schedule-frequency-value-container">
                    <div class="schedule-frequency-custom-unit-container" v-if="currentOption === ScheduleFrequencyOption.Custom">
                        <v-chip-group column mandatory selected-class="text-primary"
                                      :model-value="frequencyType"
                                      @update:model-value="changeCustomIntervalUnit">
                            <v-chip size="small" variant="outlined"
                                    :key="unit.type" :value="unit.type" :text="unit.displayName"
                                    v-for="unit in allCustomIntervalUnits" />
                        </v-chip-group>
                    </div>
                    <v-list ref="frequencyValueList" :selectable="hasFrequencyValues"
                            :select-strategy="currentOption === ScheduleFrequencyOption.Custom ? 'single-leaf' : 'classic'"
                            v-model:selected="frequencyValue" @keydown.capture="onListKeydown($event, 'right')">
                        <v-list-item :class="{ 'frequency-value-selected v-list-item--active text-primary': hasFrequencyValues && isFrequencyValueSelected(item.type) }"
                                     :key="item.type" :value="hasFrequencyValues ? item.type : undefined" :title="item.displayName"
                                     :tabindex="hasFrequencyValues ? -2 : undefined"
                                     v-for="item in frequencyItems">
                            <template #prepend="{ isSelected }" v-if="hasFrequencyValues">
                                <svg style="margin-inline: -6px 6px" width="24" height="24" viewBox="0 0 24 24" aria-hidden="true"
                                     :class="{ 'text-primary': isSelected }">
                                    <path :d="isSelected ? mdiCheckboxMarked : mdiCheckboxBlankOutline" fill="currentColor"/>
                                </svg>
                            </template>
                        </v-list-item>
                    </v-list>
                </div>
            </div>
        </template>
    </v-select>
</template>

<script setup lang="ts">
import { VSelect } from 'vuetify/components/VSelect';
import { VList } from 'vuetify/components/VList';
import { VChipGroup } from 'vuetify/components/VChipGroup';

import { ref, computed, nextTick, useTemplateRef } from 'vue';

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

import { type TypeAndDisplayName } from '@/core/base.ts';
import { type WeekDayValue } from '@/core/datetime.ts';
import { ScheduledTemplateFrequencyType } from '@/core/template.ts';

import { sortNumbersArray } from '@/lib/common.ts';
import { scrollToSelectedItem } from '@/lib/ui/common.ts';
import { type TwoColumnListColumn, setTwoColumnItemFocusWhenKeydown } from '@/lib/ui/desktop.ts';

import {
    mdiCheckboxMarked,
    mdiCheckboxBlankOutline
} from '@mdi/js';

const props = defineProps<CommonScheduleFrequencySelectionProps>();
const emit = defineEmits<{
    (e: 'update:type', value: number): void;
    (e: 'update:modelValue', value: string): void;
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

const dropdownMenu = useTemplateRef<HTMLElement>('dropdownMenu');
const frequencySelect = useTemplateRef<VSelect>('frequencySelect');
const frequencyTypeList = useTemplateRef<VList>('frequencyTypeList');
const frequencyValueList = useTemplateRef<VList>('frequencyValueList');

const menuState = ref<boolean>(false);
const customSelected = ref<boolean>(false);

const firstDayOfWeek = computed<WeekDayValue>(() => userStore.currentUserFirstDayOfWeek);

const allFrequencyOptions = computed<ScheduleFrequencyOptionItem[]>(() => getAllFrequencyOptions(props.disabledText));

const frequencyType = computed<number>(() => props.type);

const frequencyValue = computed<number[]>({
    get: () => getFrequencyValues(props.modelValue),
    set: (value: number[]) => {
        // the custom interval always needs a value, so it cannot be unselected
        if (!value.length && ScheduledTemplateFrequencyType.isEveryNInterval(props.type)) {
            return;
        }

        emit('update:modelValue', sortNumbersArray(value).join(','));
    }
});

const currentOption = computed<ScheduleFrequencyOption>(() => {
    // a custom interval which equals a preset still shows the custom builder while it is being edited
    if (customSelected.value && ScheduledTemplateFrequencyType.isEveryNInterval(frequencyType.value)) {
        return ScheduleFrequencyOption.Custom;
    }

    return getScheduleFrequencyOption(frequencyType.value, frequencyValue.value);
});

const hasFrequencyValues = computed<boolean>(() => currentOption.value === ScheduleFrequencyOption.Custom
    || currentOption.value === ScheduleFrequencyOption.DaysOfWeek
    || currentOption.value === ScheduleFrequencyOption.DaysOfMonth
    || currentOption.value === ScheduleFrequencyOption.DaysOfYear);

const displayFrequency = computed<string>(() => getFrequencyDisplayText(frequencyType.value, props.modelValue, props.disabledText));

const frequencyItems = computed<TypeAndDisplayName[]>(() => {
    switch (currentOption.value) {
        case ScheduleFrequencyOption.Disabled:
            return [{ type: 0, displayName: tt('None') }];
        case ScheduleFrequencyOption.Custom:
            return getCustomIntervalItems(frequencyType.value);
        case ScheduleFrequencyOption.DaysOfWeek:
            return allWeekDays.value;
        case ScheduleFrequencyOption.DaysOfMonth:
            return allAvailableMonthDays.value;
        case ScheduleFrequencyOption.DaysOfYear:
            return allAvailableMonthAndDays.value;
        default:
            return [{ type: 0, displayName: displayFrequency.value }];
    }
});

function updateFrequency(type: number, values: number[]): void {
    if (props.type !== type) {
        emit('update:type', type);
    }

    emit('update:modelValue', sortNumbersArray(values).join(','));
}

function selectOption(option: ScheduleFrequencyOption): void {
    const preset = getScheduleFrequencyPreset(option);
    customSelected.value = option === ScheduleFrequencyOption.Custom;

    if (preset) {
        updateFrequency(preset.type, [preset.value]);
    } else if (option === ScheduleFrequencyOption.Disabled) {
        updateFrequency(ScheduledTemplateFrequencyType.Disabled.type, []);
    } else if (option === ScheduleFrequencyOption.Custom && !ScheduledTemplateFrequencyType.isEveryNInterval(frequencyType.value)) {
        updateFrequency(ScheduledTemplateFrequencyType.EveryNDays.type, [1]);
    } else if (option === ScheduleFrequencyOption.DaysOfWeek && frequencyType.value !== ScheduledTemplateFrequencyType.Weekly.type) {
        updateFrequency(ScheduledTemplateFrequencyType.Weekly.type, [firstDayOfWeek.value]);
    } else if (option === ScheduleFrequencyOption.DaysOfMonth && frequencyType.value !== ScheduledTemplateFrequencyType.Monthly.type) {
        updateFrequency(ScheduledTemplateFrequencyType.Monthly.type, [1]);
    } else if (option === ScheduleFrequencyOption.DaysOfYear && frequencyType.value !== ScheduledTemplateFrequencyType.Yearly.type) {
        updateFrequency(ScheduledTemplateFrequencyType.Yearly.type, [101]);
    }
}

function changeCustomIntervalUnit(type: number): void {
    if (ScheduledTemplateFrequencyType.isEveryNInterval(type)) {
        updateFrequency(type, [frequencyValue.value[0] || 1]);
    }
}

function onListKeydown(event: KeyboardEvent, column: TwoColumnListColumn): void {
    setTwoColumnItemFocusWhenKeydown(event, column, {
        getList: column => {
            if (column === 'left') {
                return frequencyTypeList.value?.$el;
            } else if (column === 'right') {
                return frequencyValueList.value?.$el;
            } else {
                return null;
            }
        },
        getSelectedIndex: column => {
            if (column === 'left') {
                return allFrequencyOptions.value.findIndex(item => item.option === currentOption.value);
            } else if (column === 'right') {
                return frequencyItems.value.findIndex(item => isFrequencyValueSelected(item.type));
            } else {
                return -1;
            }
        },
        onSelect: (column, index) => {
            if (column === 'left') {
                const item = allFrequencyOptions.value[index];

                if (item) {
                    selectOption(item.option);
                }

                return;
            }

            const item = frequencyItems.value[index];

            if (item && hasFrequencyValues.value) {
                const value = item.type;

                if (!isFrequencyValueSelected(value)) {
                    if (currentOption.value === ScheduleFrequencyOption.Custom) {
                        frequencyValue.value = [value];
                    } else {
                        frequencyValue.value = [...frequencyValue.value, value];
                    }
                }
            }
        },
        onLeave: () => {
            menuState.value = false;
            frequencySelect.value?.focus();
        }
    });
}

function isFrequencyValueSelected(currentValue: number): boolean {
    return frequencyValue.value.indexOf(currentValue) >= 0;
}

function onMenuStateChanged(state: boolean): void {
    if (state) {
        customSelected.value = false;

        nextTick(() => {
            if (dropdownMenu.value && dropdownMenu.value.parentElement) {
                scrollToSelectedItem(dropdownMenu.value.parentElement, '.schedule-frequency-value-container', '.schedule-frequency-value-container', '.frequency-value-selected');
            }
        });
    }
}
</script>

<style>
.schedule-frequency-select-menu {
    max-height: inherit !important;
}

.schedule-frequency-select-menu > .v-list {
    padding: 0;
}

.schedule-frequency-select-menu .schedule-frequency-container {
    width: 100%;
    display: flex;
}

.schedule-frequency-select-menu .schedule-frequency-type-container,
.schedule-frequency-select-menu .schedule-frequency-value-container {
    width: 100%;
    max-height: 310px;
    overflow-y: scroll;
}

.schedule-frequency-select-menu .schedule-frequency-custom-unit-container {
    padding: 4px 12px 0 12px;
}
</style>
