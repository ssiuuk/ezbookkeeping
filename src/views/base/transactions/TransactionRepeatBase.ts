import { ref, computed, watch, type Ref } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useTransactionTemplatesStore } from '@/stores/transactionTemplate.ts';

import { TransactionType } from '@/core/transaction.ts';
import { TemplateType, ScheduledTemplateFrequencyType } from '@/core/template.ts';

import { Transaction } from '@/models/transaction.ts';
import { TransactionTemplate } from '@/models/transaction_template.ts';

import {
    getBrowserTimezoneName,
    getCurrentUnixTime,
    parseDateTimeFromUnixTimeWithTimezoneOffset
} from '@/lib/datetime.ts';
import { generateRandomUUID } from '@/lib/misc.ts';
import { isUserScheduledTransactionEnabled } from '@/lib/server_settings.ts';
import { getRepeatScheduledDates } from '@/lib/template.ts';
import logger from '@/lib/logger.ts';

import { TransactionEditPageMode, TransactionEditPageType } from './TransactionEditPageBase.ts';

const TEMPLATE_NAME_MAX_LENGTH: number = 64;

export const REPEAT_NOT_SET_UP_MESSAGE: string = 'The transaction has been saved, but the repeat could not be set up';

// lets a new transaction also create a scheduled transaction template which repeats it
export function useTransactionRepeatBase(pageType: TransactionEditPageType, mode: Ref<TransactionEditPageMode>, transaction: Ref<Transaction | TransactionTemplate>) {
    const { tt } = useI18n();

    const settingsStore = useSettingsStore();
    const transactionCategoriesStore = useTransactionCategoriesStore();
    const transactionTemplatesStore = useTransactionTemplatesStore();

    const repeatFrequencyType = ref<number>(ScheduledTemplateFrequencyType.Disabled.type);
    const repeatFrequency = ref<string>('');

    const showRepeatField = computed<boolean>(() => pageType === TransactionEditPageType.Transaction
        && mode.value === TransactionEditPageMode.Add
        && transaction.value.type !== TransactionType.ModifyBalance
        && isUserScheduledTransactionEnabled());

    function resetRepeat(): void {
        repeatFrequencyType.value = ScheduledTemplateFrequencyType.Disabled.type;
        repeatFrequency.value = '';
    }

    function getRepeatTemplateName(current: Transaction): string {
        let name = current.comment ? current.comment.trim() : '';

        if (!name) {
            name = transactionCategoriesStore.allTransactionCategoriesMap[current.getCategoryId()]?.name ?? '';
        }

        if (!name) {
            name = tt('Repeat');
        }

        return name.substring(0, TEMPLATE_NAME_MAX_LENGTH);
    }

    // builds the template before the transaction is saved, because the form may be reset right after saving
    function buildRepeatTemplate(): TransactionTemplate | null {
        if (!showRepeatField.value || repeatFrequencyType.value === ScheduledTemplateFrequencyType.Disabled.type || !repeatFrequency.value) {
            return null;
        }

        const current = transaction.value;
        const template = TransactionTemplate.createNewTransactionTemplate(current);

        template.templateType = TemplateType.Schedule.type;
        template.name = getRepeatTemplateName(current);
        template.tagIds = [...current.tagIds];
        template.timeZone = current.timeZone || settingsStore.appSettings.timeZone || getBrowserTimezoneName();
        template.utcOffset = current.utcOffset;
        template.scheduledFrequencyType = repeatFrequencyType.value;
        template.scheduledFrequency = repeatFrequency.value;
        const repeatDates = getRepeatScheduledDates(parseDateTimeFromUnixTimeWithTimezoneOffset(current.time, current.utcOffset));
        template.scheduledStartDate = repeatDates.startDate;
        template.scheduledSkipDate = repeatDates.skipDate;

        return template;
    }

    // creates the template in the same book as the transaction, so it must be called before the entry book is restored
    // resolves false instead of rejecting when it fails, because the transaction itself has already been saved
    function saveRepeatTemplate(template: TransactionTemplate | null): Promise<boolean> {
        if (!template) {
            return Promise.resolve(true);
        }

        return transactionTemplatesStore.saveTemplateContent({
            template: template,
            isEdit: false,
            clientSessionId: generateRandomUUID()
        }).then(() => {
            // the cached list may belong to another book, so let it reload next time
            transactionTemplatesStore.updateTransactionTemplateListInvalidState(TemplateType.Schedule.type, true);
            return true;
        }).catch(error => {
            logger.error('failed to set up repeat for the new transaction', error);
            return false;
        });
    }

    // the every N frequency types require a start date, so fill today when a scheduled template switches to one of them
    watch(() => transaction.value instanceof TransactionTemplate ? transaction.value.scheduledFrequencyType : undefined, (newType, oldType) => {
        const template = transaction.value;

        if (pageType !== TransactionEditPageType.Template || mode.value === TransactionEditPageMode.View
            || !(template instanceof TransactionTemplate) || template.templateType !== TemplateType.Schedule.type
            || newType === oldType || !newType || !ScheduledTemplateFrequencyType.isEveryNInterval(newType)
            || template.scheduledStartDate) {
            return;
        }

        template.scheduledStartDate = parseDateTimeFromUnixTimeWithTimezoneOffset(getCurrentUnixTime(), template.utcOffset).getGregorianCalendarYearDashMonthDashDay();
    });

    return {
        // states
        repeatFrequencyType,
        repeatFrequency,
        // computed states
        showRepeatField,
        // functions
        resetRepeat,
        buildRepeatTemplate,
        saveRepeatTemplate
    };
}
