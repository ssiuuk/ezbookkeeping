import { describe, expect, it } from 'vitest';

import { parseBigDecimal } from '@/lib/numeral.ts';
import { calculateMonthlyBudgetProgress } from '@/lib/budget.ts';

describe('calculateMonthlyBudgetProgress', () => {
    it('should return undefined when no budget is set', () => {
        expect(calculateMonthlyBudgetProgress(parseBigDecimal(0), parseBigDecimal(1000), parseBigDecimal(3000), 10, 30)).toBeUndefined();
        expect(calculateMonthlyBudgetProgress(parseBigDecimal(-100), parseBigDecimal(1000), parseBigDecimal(3000), 10, 30)).toBeUndefined();
    });

    it('should calculate progress when spending is on track', () => {
        const progress = calculateMonthlyBudgetProgress(parseBigDecimal(300000), parseBigDecimal(50000), parseBigDecimal(150000), 10, 30);

        expect(progress).toBeDefined();
        expect(progress?.usedRate).toBeCloseTo(50000 / 300000);
        expect(progress?.remainingAmount.toString()).toBe('250000');
        expect(progress?.overBudget).toBe(false);
        expect(progress?.aheadOfSchedule).toBe(false);
        expect(progress?.dailyAllowance.toString()).toBe('11904.761904761904762');
        expect(progress?.projectedOverBudget).toBe(false);
    });

    it('should flag spending that is ahead of schedule and projected to exceed the budget', () => {
        const progress = calculateMonthlyBudgetProgress(parseBigDecimal(300000), parseBigDecimal(200000), parseBigDecimal(600000), 10, 30);

        expect(progress?.overBudget).toBe(false);
        expect(progress?.aheadOfSchedule).toBe(true);
        expect(progress?.projectedOverBudget).toBe(true);
    });

    it('should report zero daily allowance once the budget is exceeded', () => {
        const progress = calculateMonthlyBudgetProgress(parseBigDecimal(300000), parseBigDecimal(350000), parseBigDecimal(500000), 21, 30);

        expect(progress?.usedRate).toBeCloseTo(350000 / 300000);
        expect(progress?.remainingAmount.toString()).toBe('-50000');
        expect(progress?.overBudget).toBe(true);
        expect(progress?.dailyAllowance.isZero()).toBe(true);
    });

    it('should spread the remaining budget over today on the last day of the month', () => {
        const progress = calculateMonthlyBudgetProgress(parseBigDecimal(300000), parseBigDecimal(290000), parseBigDecimal(290000), 31, 31);

        expect(progress?.dailyAllowance.toString()).toBe('10000');
    });
});
