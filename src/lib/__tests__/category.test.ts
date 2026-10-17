import { describe, expect, it } from 'vitest';

import { CategoryType } from '@/core/category.ts';
import { TransactionCategory } from '@/models/transaction_category.ts';
import { getParentCategoryIdForNewSubCategory } from '@/lib/category.ts';

function createCategory(id: string, parentId: string, visible: boolean, subCategories?: TransactionCategory[]): TransactionCategory {
    const category = TransactionCategory.createNewCategory(CategoryType.Expense, parentId);
    category.id = id;
    category.visible = visible;

    if (subCategories) {
        category.subCategories = subCategories;
    }

    return category;
}

function createAllCategories(): Record<number, TransactionCategory[]> {
    return {
        [CategoryType.Expense]: [
            createCategory('food', '0', true, [
                createCategory('food-lunch', 'food', true)
            ]),
            createCategory('archived', '0', false, [
                createCategory('archived-sub', 'archived', true)
            ]),
            createCategory('transport', '0', true, [
                createCategory('transport-bus', 'transport', true)
            ])
        ]
    };
}

describe('getParentCategoryIdForNewSubCategory', () => {
    it('should prefer the given primary category when it is visible', () => {
        expect(getParentCategoryIdForNewSubCategory(createAllCategories(), CategoryType.Expense, 'transport', 'food-lunch')).toBe('transport');
    });

    it('should fall back to the primary category of the selected category', () => {
        expect(getParentCategoryIdForNewSubCategory(createAllCategories(), CategoryType.Expense, undefined, 'transport-bus')).toBe('transport');
    });

    it('should ignore hidden primary categories', () => {
        expect(getParentCategoryIdForNewSubCategory(createAllCategories(), CategoryType.Expense, 'archived', 'archived-sub')).toBe('food');
    });

    it('should fall back to the first visible primary category', () => {
        expect(getParentCategoryIdForNewSubCategory(createAllCategories(), CategoryType.Expense)).toBe('food');
    });

    it('should return undefined when there is no primary category', () => {
        expect(getParentCategoryIdForNewSubCategory(createAllCategories(), CategoryType.Income)).toBeUndefined();
    });
});
