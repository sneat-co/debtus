import { CurrencyCode } from '../services/debtus-service';

// ---------------------------------------------------------------------------
// Debtus domain model (UI-facing).
//
// These interfaces mirror the debtus Go backend domain (see
// backend/debtus/models4debtus/*.go) but are trimmed to what the web UI needs.
// Money is represented here in MAJOR units (e.g. 12.34), matching the
// create-transfer API's `amount: { currency, value }` shape. The backend
// stores fixed-point cents internally; conversion happens server-side.
// ---------------------------------------------------------------------------

/**
 * UI-facing transfer direction, from the perspective of the current user.
 * - `lend`   = "I gave" money to the counterparty (they owe me).
 * - `borrow` = "I got"  money from the counterparty (I owe them).
 * Maps to the backend `TransferDirection` enum (`u2c` / `c2u`).
 */
export type DebtDirection = 'lend' | 'borrow';

/**
 * Backend transfer direction enum (models4debtus/transfer.go).
 * `3d-party` exists in the backend but is not exposed by this UI yet.
 */
export type ApiTransferDirection = 'u2c' | 'c2u' | '3d-party';

export interface IAmount {
  readonly currency: CurrencyCode;
  readonly value: number;
}

/**
 * A per-currency signed balance with a single counterparty.
 * Positive value = the counterparty owes the current user (net lent).
 * Negative value = the current user owes the counterparty (net borrowed).
 */
export type SignedBalance = Partial<Record<CurrencyCode, number>>;

/**
 * A counterparty (reuses a contactus space contact — `contactID` is a
 * contactus contact id) together with the current user's net balance.
 *
 * Cross-space support: `counterpartySpaceID` names the space the counterparty
 * belongs to. When it differs from the current (creditor) space, the loan
 * crosses a space boundary.
 */
export interface IContactBalance {
  readonly contactID: string;
  readonly title: string;
  readonly balance: SignedBalance;
  /** The space the counterparty contact belongs to (for cross-space loans). */
  readonly counterpartySpaceID?: string;
}

export interface IDebtusTransfer {
  readonly id: string;
  readonly direction: DebtDirection;
  readonly amount: IAmount;
  readonly counterpartyContactID: string;
  readonly counterpartyTitle: string;
  readonly note?: string;
  readonly created: string; // ISO date
  readonly dueOn?: string; // ISO date
  readonly isReturn: boolean;
  readonly isOutstanding: boolean;
  /** Space the creator (current user) side belongs to. */
  readonly creatorSpaceID: string;
  /** Space the counterparty side belongs to (may differ — cross-space loan). */
  readonly counterpartySpaceID?: string;
}

// ---- create-transfer request/response (POST /api4debtus/create-transfer) ----

export interface ICreateTransferRequest {
  readonly spaceID: string;
  readonly direction: DebtDirection;
  readonly amount: IAmount;
  /** contactus contact id of the counterparty. */
  readonly contactID: string;
  /** Display name; used when creating a brand-new counterparty. */
  readonly contactTitle?: string;
  readonly note?: string;
  readonly dueOn?: string; // ISO date
  /** Settle-up flag — reverses direction and nets against the balance. */
  readonly isReturn?: boolean;
  readonly returnToTransferID?: string;
  /**
   * Cross-space: the counterparty's own space, when it differs from `spaceID`.
   * NOTE: the current backend create-transfer body accepts a single `spaceID`
   * plus from/to contact ids; representing the counterparty space explicitly
   * still needs backend verification (tracked in the PR).
   */
  readonly counterpartySpaceID?: string;
}

export interface ICreateTransferResponse {
  readonly transfer: IDebtusTransfer;
  readonly userBalance: SignedBalance;
  readonly counterpartyBalance: SignedBalance;
}

export interface ISettleUpRequest {
  readonly spaceID: string;
  readonly contactID: string;
  readonly contactTitle?: string;
  readonly amount: IAmount;
  readonly counterpartySpaceID?: string;
}
