import { describe, expect, test } from 'vitest';

import type { TextualYearMonthDay } from '@/core/datetime.ts';
import { TransactionType } from '@/core/transaction.ts';
import { TemplateType, ScheduledTemplateFrequencyType } from '@/core/template.ts';

import { TransactionTemplate, type TransactionTemplateInfoResponse } from '@/models/transaction_template.ts';

function scheduledTemplateResponse(scheduledSkipDate?: TextualYearMonthDay): TransactionTemplateInfoResponse {
    return {
        id: '1',
        timeSequenceId: '1',
        type: TransactionType.Expense,
        categoryId: '2',
        time: 0,
        utcOffset: 480,
        sourceAccountId: '3',
        destinationAccountId: '0',
        sourceAmount: 100,
        destinationAmount: 0,
        hideAmount: false,
        tagIds: [],
        comment: '',
        editable: true,
        templateType: TemplateType.Schedule.type,
        name: 'Rent',
        scheduledFrequencyType: ScheduledTemplateFrequencyType.EveryNMonths.type,
        scheduledFrequency: '1',
        scheduledStartDate: '2027-01-31',
        scheduledSkipDate: scheduledSkipDate,
        scheduledAt: 960,
        timeZone: 'Asia/Shanghai',
        displayOrder: 1,
        hidden: false
    };
}

describe('transaction template', () => {
    test('keeps the skip date when a scheduled template is edited', () => {
        const template = TransactionTemplate.ofTemplate(scheduledTemplateResponse('2027-01-31'));
        expect(template.scheduledStartDate).toBe('2027-01-31');
        expect(template.scheduledSkipDate).toBe('2027-01-31');
        expect(template.toTemplateModifyRequest().scheduledSkipDate).toBe('2027-01-31');
        expect(template.toTemplateCreateRequest('session').scheduledSkipDate).toBe('2027-01-31');

        const copied = TransactionTemplate.ofTemplate(scheduledTemplateResponse());
        copied.fillFrom(template);
        expect(copied.scheduledSkipDate).toBe('2027-01-31');
    });

    test('does not send a skip date when the template has none', () => {
        const template = TransactionTemplate.ofTemplate(scheduledTemplateResponse());
        expect(template.scheduledSkipDate).toBeUndefined();
        expect(template.toTemplateModifyRequest().scheduledSkipDate).toBeUndefined();
    });
});
