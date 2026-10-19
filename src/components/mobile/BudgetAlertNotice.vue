<template>
    <f7-list strong inset dividers class="budget-alert-notice" v-if="noticeItems.length">
        <f7-list-item :key="item.alert.id"
                      :title="getNoticeText(item)"
                      v-for="item in noticeItems">
            <template #media>
                <f7-icon :class="item.exceeded ? 'text-color-red' : 'text-color-orange'" f7="exclamationmark_triangle_fill"></f7-icon>
            </template>
            <template #after>
                <f7-link icon-f7="xmark" :class="{ 'disabled': dismissing }" :aria-label="tt('Dismiss')" @click="dismiss(item)"></f7-link>
            </template>
        </f7-list-item>
        <f7-list-button :class="{ 'disabled': dismissing }" :title="tt('Dismiss All')" @click="dismissAll" v-if="noticeItems.length > 1"></f7-list-button>
    </f7-list>
</template>

<script setup lang="ts">
import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents } from '@/lib/ui/mobile.ts';
import { useBudgetAlertNoticeBase } from '@/views/base/budgets/BudgetAlertNoticeBase.ts';

import type { BudgetAlertNoticeItem } from '@/lib/budget_alert.ts';
import logger from '@/lib/logger.ts';

const { tt } = useI18n();
const { showToast } = useI18nUIComponents();
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
            showToast(error.message || error);
        }
    });
}

function dismissAll(): void {
    dismissAllNotices().catch(error => {
        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

reload();

defineExpose({
    reload
});
</script>

<style>
.budget-alert-notice .item-title {
    white-space: normal;
}
</style>
