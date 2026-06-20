import { Injectable, inject } from '@angular/core';
import { SneatApiService } from '@sneat/api';
import {
  ICreateSplitRecordRequest,
  ISplitusService,
} from '@sneat/extension-splitus-contract';
import { Observable } from 'rxjs';

@Injectable()
export class SplitusService implements ISplitusService {
  private readonly sneatApiService = inject(SneatApiService);

  public createSplitRecord(
    request: ICreateSplitRecordRequest,
  ): Observable<string> {
    return this.sneatApiService.post('splitus/create_split_record', request);
  }
}
