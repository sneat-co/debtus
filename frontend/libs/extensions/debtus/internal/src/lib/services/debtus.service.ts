import { Injectable, inject } from '@angular/core';
import { SneatApiService } from '@sneat/api';
import {
  ICreateDebtRecordRequest,
  IDebtusService,
} from '@sneat/extension-debtus-contract';
import { Observable } from 'rxjs';

@Injectable()
export class DebtusService implements IDebtusService {
  private readonly sneatApiService = inject(SneatApiService);

  public createDebtRecord(
    request: ICreateDebtRecordRequest,
  ): Observable<string> {
    return this.sneatApiService.post('debtus/create_debt_record', request);
  }
}
