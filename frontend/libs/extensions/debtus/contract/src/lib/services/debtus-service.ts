import { InjectionToken } from '@angular/core';
import { Observable } from 'rxjs';

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
  createDebtRecord(request: ICreateDebtRecordRequest): Observable<string>;
}

export const DEBTUS_SERVICE = new InjectionToken<IDebtusService>('DebtusService');
