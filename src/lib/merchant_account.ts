// finds the shop name as it is already used in the book, ignoring case and surrounding spaces
export function findKnownMerchant(knownMerchants: readonly string[], merchant: string): string | undefined {
    const name = merchant.trim().toLowerCase();
    return name ? knownMerchants.find(item => item.trim().toLowerCase() === name) : undefined;
}

export interface MerchantAccountLookupRequest {
    readonly merchant: string;
    readonly knownMerchants: readonly string[];
    // the shop name in the editor when the answer arrives, so an answer for a shop that was changed since goes unused
    readonly currentMerchant: () => string;
    // whether the editor is still the one that asked (same session, still adding, account not picked by hand)
    readonly isStillCurrent: () => boolean;
    readonly apply: (accountId: string) => void;
}

// looks up the account last used at a shop; every new lookup and every cancel makes an answer still on its way go unused
export class MerchantAccountLookup {
    private requestId: number = 0;

    public constructor(private readonly fetchAccountId: (merchant: string) => Promise<string | undefined>) {
    }

    public cancel(): void {
        this.requestId++;
    }

    public lookup(request: MerchantAccountLookupRequest): Promise<void> {
        const requestId = ++this.requestId;
        const merchant = findKnownMerchant(request.knownMerchants, request.merchant);

        if (!merchant) {
            return Promise.resolve();
        }

        return this.fetchAccountId(merchant).then(accountId => {
            if (requestId !== this.requestId || !accountId || accountId === '0') {
                return;
            }

            if (findKnownMerchant([merchant], request.currentMerchant()) !== merchant || !request.isStillCurrent()) {
                return;
            }

            request.apply(accountId);
        }).catch(() => {
            // the shop's account is only a convenience, so a failed lookup leaves the account as it is
        });
    }
}
