<template>
    <div class="budget-alert-notice" v-if="noticeItems.length">
        <v-alert class="mb-2" density="compact" variant="tonal"
                 :type="item.exceeded ? 'error' : 'warning'"
                 :key="item.alert.id"
                 v-for="item in noticeItems">
            <div class="d-flex align-center">
                <span>{{ getNoticeText(item) }}</span>
                <v-spacer/>
                <v-btn class="ms-2" density="compact" variant="text" :icon="mdiClose" size="small"
                       :aria-label="tt('Dismiss')" :disabled="dismissing" @click="dismiss(item)"/>
            </div>
        </v-alert>
        <div class="d-flex justify-end mb-2" v-if="noticeItems.length > 1">
            <v-btn density="compact" variant="text" :disabled="dismissing" @click="dismissAll">{{ tt('Dismiss All') }}</v-btn>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useI18n } from '@/locales/helpers.ts';
import { useBudgetAlertNoticeBase } from '@/views/base/budgets/BudgetAlertNoticeBase.ts';

import type { ErrorResponse } from '@/core/api.ts';
import type { BudgetAlertNoticeItem } from '@/lib/budget_alert.ts';
import logger from '@/lib/logger.ts';

import {
    mdiClose
} from '@mdi/js';

const emit = defineEmits<{
    (e: 'error', error: string | { message: string } | { error: ErrorResponse }): void;
}>();

const { tt } = useI18n();
const {
    dismissing,
    noticeItems,
    getNoticeText,
    loadAlerts,
    dismissNotice,
    dismissAllNotices
} = useBudgetAlertNoticeBase();

function reload(): void {
    // the notices are optional, so a failure of loading them is only logged
    loadAlerts().catch(error => {
        logger.warn('failed to load budget alerts', error);
    });
}

function dismiss(item: BudgetAlertNoticeItem): void {
    dismissNotice(item).catch(error => {
        if (!error.processed) {
            emit('error', error);
        }
    });
}

function dismissAll(): void {
    dismissAllNotices().catch(error => {
        if (!error.processed) {
            emit('error', error);
        }
    });
}

reload();

defineExpose({
    reload
});
</script>
