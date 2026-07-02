import { HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { SneatApiService } from '@sneat/api';
import {
  CurrencyCode,
  ICreateSplitRequest,
  ICreateSplitResponse,
  ISplit,
  ISplitListItem,
  ISplitParticipant,
  ISplitusService,
  SplitShareStatus,
} from '@sneat/extension-splitus-contract';
import { Observable, map } from 'rxjs';

// Backend DTO shapes (backend/splitus/api4splitusbot/api_create_split.go,
// api_get_splits.go). decimal.Decimal64p2 fields (amount/share) marshal as
// plain integer CENTS (e.g. 9000 == "90.00" — see strongo/decimal
// MarshalJSON), unlike the create-split REQUEST's `amount`/shares, which the
// Go DTO types as decimal strings ("90.00") and are sent through unchanged.
interface IApiCreateSplitTransferDto {
  readonly id: string;
  readonly contactID: string;
  readonly amount: number; // cents
}

interface IApiCreateSplitResponseDto {
  readonly id: string;
  readonly transfers: IApiCreateSplitTransferDto[];
}

interface IApiSplitParticipantDto {
  readonly contactID?: string;
  readonly userID?: string;
  readonly name: string;
  readonly share: number; // cents
  readonly isPayer?: boolean;
  readonly status: SplitShareStatus;
}

interface IApiGetSplitResponseDto {
  readonly id: string;
  readonly title?: string;
  readonly currency: CurrencyCode;
  readonly amount: number; // cents
  readonly status: string;
  readonly participants: IApiSplitParticipantDto[];
}

interface IApiSplitListItemDto {
  readonly id: string;
  readonly title?: string;
  readonly amount: number; // cents
  readonly currency: CurrencyCode;
  readonly status: string;
  readonly membersCount: number;
}

interface IApiGetSplitsResponseDto {
  readonly splits: IApiSplitListItemDto[];
}

/** Converts the backend's fixed-point cents (e.g. 9000) to major units (90). */
const centsToAmount = (cents: number): number => cents / 100;

@Injectable()
export class SplitusService implements ISplitusService {
  private readonly sneatApiService = inject(SneatApiService);

  /** REAL: POST /api4splitus/create-split. Payer is the authenticated user. */
  public createSplit(
    request: ICreateSplitRequest,
  ): Observable<ICreateSplitResponse> {
    const body = {
      spaceID: request.spaceID,
      title: request.title,
      currency: request.currency,
      amount: request.amount,
      splitMode: request.splitMode,
      participantContactIDs: request.participantContactIDs,
      shares: request.shares,
    };
    return this.sneatApiService
      .post<IApiCreateSplitResponseDto>('api4splitus/create-split', body)
      .pipe(map((dto) => this.mapCreateSplitResponse(dto)));
  }

  /** REAL: GET /api4splitus/split?spaceID=&id= */
  public getSplit(spaceID: string, id: string): Observable<ISplit> {
    const params = new HttpParams().set('spaceID', spaceID).set('id', id);
    return this.sneatApiService
      .get<IApiGetSplitResponseDto>('api4splitus/split', params)
      .pipe(map((dto) => this.mapSplit(dto)));
  }

  /** REAL: GET /api4splitus/splits?spaceID= */
  public getSplits(spaceID: string): Observable<ISplitListItem[]> {
    const params = new HttpParams().set('spaceID', spaceID);
    return this.sneatApiService
      .get<IApiGetSplitsResponseDto>('api4splitus/splits', params)
      .pipe(map((dto) => dto.splits.map((s) => this.mapSplitListItem(s))));
  }

  private mapCreateSplitResponse(
    dto: IApiCreateSplitResponseDto,
  ): ICreateSplitResponse {
    return {
      id: dto.id,
      transfers: (dto.transfers ?? []).map((t) => ({
        id: t.id,
        contactID: t.contactID,
        amount: centsToAmount(t.amount),
      })),
    };
  }

  private mapSplit(dto: IApiGetSplitResponseDto): ISplit {
    return {
      id: dto.id,
      title: dto.title,
      currency: dto.currency,
      amount: centsToAmount(dto.amount),
      status: dto.status,
      participants: (dto.participants ?? []).map((p) =>
        this.mapParticipant(p),
      ),
    };
  }

  // Settled/outstanding is read verbatim off the API response — Splitus
  // never computes or caches settled state on the client (see
  // handleGetSplit's Debtus read-through in api_get_splits.go).
  private mapParticipant(dto: IApiSplitParticipantDto): ISplitParticipant {
    return {
      contactID: dto.contactID,
      userID: dto.userID,
      name: dto.name,
      share: centsToAmount(dto.share),
      isPayer: dto.isPayer,
      status: dto.status,
    };
  }

  private mapSplitListItem(dto: IApiSplitListItemDto): ISplitListItem {
    return {
      id: dto.id,
      title: dto.title,
      amount: centsToAmount(dto.amount),
      currency: dto.currency,
      status: dto.status,
      membersCount: dto.membersCount,
    };
  }
}
