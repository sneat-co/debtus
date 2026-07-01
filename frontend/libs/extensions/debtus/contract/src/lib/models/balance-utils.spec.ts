import { describe, expect, it } from 'vitest';
import { IContactBalance } from './debtus-models';
import {
  apiDirectionToDebtDirection,
  debtDirectionToApiDirection,
  formatAmount,
  formatSignedBalance,
  isZeroBalance,
  reverseApiDirection,
  round2,
  settleDirectionForBalance,
  signedTransferDelta,
  summarizeBalances,
} from './balance-utils';

describe('direction mapping', () => {
  it('maps UI direction to API direction', () => {
    expect(debtDirectionToApiDirection('lend')).toBe('u2c');
    expect(debtDirectionToApiDirection('borrow')).toBe('c2u');
  });

  it('maps API direction to UI direction', () => {
    expect(apiDirectionToDebtDirection('u2c')).toBe('lend');
    expect(apiDirectionToDebtDirection('c2u')).toBe('borrow');
    expect(apiDirectionToDebtDirection('3d-party')).toBe('lend');
  });

  it('round-trips lend/borrow', () => {
    for (const d of ['lend', 'borrow'] as const) {
      expect(apiDirectionToDebtDirection(debtDirectionToApiDirection(d))).toBe(
        d,
      );
    }
  });

  it('reverses API direction for returns/settle', () => {
    expect(reverseApiDirection('u2c')).toBe('c2u');
    expect(reverseApiDirection('c2u')).toBe('u2c');
    expect(reverseApiDirection('3d-party')).toBe('3d-party');
  });
});

describe('signedTransferDelta', () => {
  it('lend increases what they owe (positive)', () => {
    expect(signedTransferDelta('lend', 100)).toBe(100);
  });
  it('borrow increases what you owe (negative)', () => {
    expect(signedTransferDelta('borrow', 100)).toBe(-100);
  });
});

describe('settleDirectionForBalance', () => {
  it('counterparty owes you -> they return -> borrow', () => {
    expect(settleDirectionForBalance(50)).toBe('borrow');
  });
  it('you owe -> you return -> lend', () => {
    expect(settleDirectionForBalance(-50)).toBe('lend');
  });
});

describe('round2', () => {
  it('avoids float dust', () => {
    expect(round2(0.1 + 0.2)).toBe(0.3);
    expect(round2(10 / 3)).toBe(3.33);
  });
});

describe('summarizeBalances', () => {
  const contacts: IContactBalance[] = [
    { contactID: 'a', title: 'Alice', balance: { USD: 100, EUR: -20 } },
    { contactID: 'b', title: 'Bob', balance: { USD: -40 } },
    { contactID: 'c', title: 'Carol', balance: { EUR: 20 } },
  ];

  it('nets amounts by currency and splits by sign', () => {
    const summary = summarizeBalances(contacts);
    // USD: 100 - 40 = 60 owed to you; EUR: -20 + 20 = 0 (dropped)
    expect(summary.theyOweYou).toEqual([{ currency: 'USD', value: 60 }]);
    expect(summary.youOwe).toEqual([]);
  });

  it('reports what you owe when net is negative', () => {
    const summary = summarizeBalances([
      { contactID: 'x', title: 'X', balance: { USD: -75 } },
    ]);
    expect(summary.youOwe).toEqual([{ currency: 'USD', value: 75 }]);
    expect(summary.theyOweYou).toEqual([]);
  });

  it('handles empty input', () => {
    expect(summarizeBalances([])).toEqual({ theyOweYou: [], youOwe: [] });
  });
});

describe('isZeroBalance', () => {
  it('true when all currencies net to zero', () => {
    expect(isZeroBalance({ USD: 0 })).toBe(true);
    expect(isZeroBalance({})).toBe(true);
  });
  it('false when any currency is non-zero', () => {
    expect(isZeroBalance({ USD: 0, EUR: 5 })).toBe(false);
  });
});

describe('formatting', () => {
  it('formats an amount to 2 decimals', () => {
    expect(formatAmount({ currency: 'USD', value: 12.5 })).toBe('12.50 USD');
  });
  it('formats a signed balance from user perspective', () => {
    expect(formatSignedBalance({ USD: 30 })).toBe('owes you 30.00 USD');
    expect(formatSignedBalance({ USD: -30 })).toBe('you owe 30.00 USD');
    expect(formatSignedBalance({ USD: 0 })).toBe('settled up');
  });
});
