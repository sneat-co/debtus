import { InjectionToken } from '@angular/core';
import { Observable } from 'rxjs';
import type {
  ICreateSplitRequest,
  ICreateSplitResponse,
  ISplit,
  ISplitListItem,
} from '../models/splitus-models';

export type CurrencyCode = 'EUR' | 'USD';

// ISplitusService is the runtime-light contract the splitus components
// depend on. Members mirror the concrete SplitusService's public surface
// exactly; the implementation lives in the -internal lib and is provided via
// the SPLITUS_SERVICE token below.
//
// Splitus persists no balance of its own — the payer/participants/shares
// given to createSplit are posted to the real backend (POST
// /api4splitus/create-split), which records the expense and posts Debtus
// transfers as the only who-owes-what records. getSplit/getSplits read the
// settled state back verbatim from that same API — there is no client-side
// settled cache (see ISplitParticipant.status in splitus-models.ts).
export interface ISplitusService {
  /** REAL: POST /api4splitus/create-split. Payer is the authenticated user. */
  createSplit(request: ICreateSplitRequest): Observable<ICreateSplitResponse>;

  /** REAL: GET /api4splitus/split?spaceID=&id= */
  getSplit(spaceID: string, id: string): Observable<ISplit>;

  /** REAL: GET /api4splitus/splits?spaceID= */
  getSplits(spaceID: string): Observable<ISplitListItem[]>;
}

export const SPLITUS_SERVICE = new InjectionToken<ISplitusService>('SplitusService');
