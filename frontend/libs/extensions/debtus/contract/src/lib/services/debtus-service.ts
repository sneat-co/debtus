import { InjectionToken } from '@angular/core';
import { Observable } from 'rxjs';
import type {
  IContactBalance,
  ICreateTransferRequest,
  ICreateTransferResponse,
  IDebtusTransfer,
  ISettleUpRequest,
} from '../models/debtus-models';

export type CurrencyCode = 'EUR' | 'USD';

export interface ICreateDebtRecordRequest {
  spaceID: string;
  contactID: string;
  currency: CurrencyCode;
  amount: number;
}

// IDebtusService is the runtime-light contract the debtus components depend on.
// Members mirror the concrete DebtusService's public surface exactly; the
// implementation lives in the -internal lib and is provided via the
// DEBTUS_SERVICE token below.
export interface IDebtusService {
  /** Legacy thin create endpoint (kept for backwards compatibility). */
  createDebtRecord(request: ICreateDebtRecordRequest): Observable<string>;

  /** Space-wide list of counterparties with their net balance. */
  getContactBalances(spaceID: string): Observable<IContactBalance[]>;

  /** Net balance with a single counterparty. */
  getContactBalance(
    spaceID: string,
    contactID: string,
  ): Observable<IContactBalance>;

  /** Transfer history, optionally filtered to one counterparty. */
  getTransfers(
    spaceID: string,
    contactID?: string,
  ): Observable<IDebtusTransfer[]>;

  /** A single transfer / receipt. */
  getTransfer(
    spaceID: string,
    transferID: string,
  ): Observable<IDebtusTransfer>;

  /** Create a transfer (lend/borrow). Also used for settle-up returns. */
  createTransfer(
    request: ICreateTransferRequest,
  ): Observable<ICreateTransferResponse>;

  /** Record a settling (return) transfer against a counterparty balance. */
  settleUp(request: ISettleUpRequest): Observable<ICreateTransferResponse>;
}

export const DEBTUS_SERVICE = new InjectionToken<IDebtusService>('DebtusService');
