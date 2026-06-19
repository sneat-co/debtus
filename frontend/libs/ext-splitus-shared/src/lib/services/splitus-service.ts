import { Injectable, inject } from '@angular/core';
import { SneatApiService } from '@sneat/api';
import { Observable } from 'rxjs';

export type CurrencyCode = 'EUR' | 'USD';

export interface ICreateSplitRecordRequest {
  spaceID: string;
  contactID: string;
  currency: CurrencyCode;
  amount: number;
}

@Injectable()
export class SplitusService {
  private readonly sneatApiService = inject(SneatApiService);

  public createSplitRecord(
    request: ICreateSplitRecordRequest,
  ): Observable<string> {
    return this.sneatApiService.post('splitus/create_split_record', request);
  }
}
