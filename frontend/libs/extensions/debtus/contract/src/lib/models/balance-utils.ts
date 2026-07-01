import { CurrencyCode } from '../services/debtus-service';
import {
  ApiTransferDirection,
  DebtDirection,
  IAmount,
  IContactBalance,
  SignedBalance,
} from './debtus-models';

// ---------------------------------------------------------------------------
// Pure, framework-free helpers for balance computation and direction mapping.
// Kept in the contract lib so both the internal impl and the UI can reuse them,
// and so they are trivially unit-testable (see balance-utils.spec.ts).
// ---------------------------------------------------------------------------

/** Map the UI direction to the backend `TransferDirection` enum. */
export function debtDirectionToApiDirection(
  direction: DebtDirection,
): ApiTransferDirection {
  return direction === 'lend' ? 'u2c' : 'c2u';
}

/** Map a backend `TransferDirection` back to the UI direction. */
export function apiDirectionToDebtDirection(
  apiDirection: ApiTransferDirection,
): DebtDirection {
  // `3d-party` is not surfaced in this UI; treat it as a lend for display.
  return apiDirection === 'c2u' ? 'borrow' : 'lend';
}

/** Reverse an API direction (used by settle-up / returns). */
export function reverseApiDirection(
  apiDirection: ApiTransferDirection,
): ApiTransferDirection {
  switch (apiDirection) {
    case 'u2c':
      return 'c2u';
    case 'c2u':
      return 'u2c';
    default:
      return apiDirection;
  }
}

/**
 * Signed delta a single transfer applies to the current user's balance with a
 * counterparty. `lend` increases what they owe the user (+), `borrow`
 * increases what the user owes (-).
 */
export function signedTransferDelta(
  direction: DebtDirection,
  value: number,
): number {
  return direction === 'lend' ? value : -value;
}

/** True when a positive signed balance means the counterparty owes the user. */
export function isOwedToUser(value: number): boolean {
  return value > 0;
}

/**
 * The direction a settle-up transfer must use to move a signed balance toward
 * zero. If the counterparty owes the user (+), they return money -> `borrow`
 * from the user's perspective (user "gets" it back). If the user owes (-), the
 * user returns money -> `lend`.
 */
export function settleDirectionForBalance(value: number): DebtDirection {
  return value > 0 ? 'borrow' : 'lend';
}

/** Round to 2 decimals, avoiding float dust (e.g. 0.1 + 0.2). */
export function round2(value: number): number {
  return Math.round((value + Number.EPSILON) * 100) / 100;
}

/**
 * Aggregate per-contact signed balances into a space-wide summary, split by
 * currency into what others owe the user vs. what the user owes others.
 * Zero net amounts (per currency) are dropped.
 */
export interface IBalanceSummary {
  readonly theyOweYou: IAmount[];
  readonly youOwe: IAmount[];
}

export function summarizeBalances(
  contacts: readonly IContactBalance[],
): IBalanceSummary {
  const totals = new Map<CurrencyCode, number>();
  for (const contact of contacts) {
    for (const [currency, value] of Object.entries(contact.balance)) {
      if (value === undefined) {
        continue;
      }
      const cc = currency as CurrencyCode;
      totals.set(cc, round2((totals.get(cc) ?? 0) + value));
    }
  }
  const theyOweYou: IAmount[] = [];
  const youOwe: IAmount[] = [];
  for (const [currency, value] of totals) {
    if (value > 0) {
      theyOweYou.push({ currency, value });
    } else if (value < 0) {
      youOwe.push({ currency, value: -value });
    }
  }
  return { theyOweYou, youOwe };
}

/** True when a contact's net balance is zero across every currency. */
export function isZeroBalance(balance: SignedBalance): boolean {
  return Object.values(balance).every((v) => !v || round2(v) === 0);
}

/** Format a money amount for display, e.g. `{ USD, 12.5 }` -> `12.50 USD`. */
export function formatAmount(amount: IAmount): string {
  return `${amount.value.toFixed(2)} ${amount.currency}`;
}

/**
 * Human-readable per-currency balance line from the user's perspective.
 * Positive -> "owes you", negative -> "you owe".
 */
export function formatSignedBalance(balance: SignedBalance): string {
  const parts: string[] = [];
  for (const [currency, value] of Object.entries(balance)) {
    if (!value || round2(value) === 0) {
      continue;
    }
    const abs = Math.abs(value).toFixed(2);
    parts.push(
      value > 0
        ? `owes you ${abs} ${currency}`
        : `you owe ${abs} ${currency}`,
    );
  }
  return parts.length ? parts.join(', ') : 'settled up';
}
