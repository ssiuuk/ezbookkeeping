// an item (category, account, tag or project) selected in an entry, to find the same item in another book
export interface BookItemReference {
    readonly id: string;
    readonly name: string;
    readonly parentName?: string;
}

interface BookItem {
    readonly id: string;
    readonly name: string;
}

function isSameName(name1: string | undefined, name2: string | undefined): boolean {
    return (name1 ?? '').trim() === (name2 ?? '').trim();
}

// returns the id of the referenced item itself, or else of the item with the same name (one under a parent with the same name first, then a visible one),
// or empty when no item has that name, so the user has to choose one
export function findSameBookItemId<T extends BookItem>(items: readonly T[], reference: BookItemReference | undefined, isVisible: (item: T) => boolean, getParentName?: (item: T) => string | undefined): string {
    if (!reference || !reference.name.trim()) {
        return '';
    }

    let foundId = '';
    let foundScore = -1;

    for (const item of items) {
        if (item.id === reference.id) {
            return item.id;
        }

        if (!isSameName(item.name, reference.name)) {
            continue;
        }

        let score = isVisible(item) ? 1 : 0;

        if (getParentName && isSameName(getParentName(item), reference.parentName)) {
            score += 2;
        }

        if (score > foundScore) {
            foundId = item.id;
            foundScore = score;
        }
    }

    return foundId;
}

// returns the ids of the items found by findSameBookItemId, skipping the references without such an item
export function findSameBookItemIds<T extends BookItem>(items: readonly T[], references: readonly BookItemReference[], isVisible: (item: T) => boolean): string[] {
    const ids: string[] = [];

    for (const reference of references) {
        const id = findSameBookItemId(items, reference, isVisible);

        if (id && ids.indexOf(id) < 0) {
            ids.push(id);
        }
    }

    return ids;
}
