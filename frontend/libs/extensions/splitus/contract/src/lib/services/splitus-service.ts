import { InjectionToken } from '@angular/core';
import { Observable } from 'rxjs';

export type CurrencyCode = 'EUR' | 'USD';

export interface ICreateSplitRecordRequest {
  spaceID: string;
  contactID: string;
  currency: CurrencyCode;
  amount: number;
}

// ISplitusService is the runtime-light contract the splitus components depend on.
// Members mirror the concrete SplitusService's public surface exactly; the
// implementation lives in the -internal lib and is provided via the
// SPLITUS_SERVICE token below.
export interface ISplitusService {
  createSplitRecord(request: ICreateSplitRecordRequest): Observable<string>;
}

export const SPLITUS_SERVICE = new InjectionToken<ISplitusService>('SplitusService');
