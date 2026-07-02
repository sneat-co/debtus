import { Injectable, inject } from '@angular/core';
import { SneatApiService } from '@sneat/api';
import {
  IContactBalance,
  ICreateDebtRecordRequest,
  ICreateTransferRequest,
  ICreateTransferResponse,
  IDebtusService,
  IDebtusTransfer,
  ISettleUpRequest,
  apiDirectionToDebtDirection,
  debtDirectionToApiDirection,
} from '@sneat/extension-debtus-contract';
import { Observable, map, of, throwError } from 'rxjs';
import { DEMO_CONTACT_BALANCES, demoTransfersForSpace } from './demo-data';

// Backend DTO shapes (facade4debtus/dto4debtus). Only the fields the UI reads
// are declared here.
interface IApiContactDto {
  readonly ID: string;
  readonly UserID?: string;
  readonly Name: string;
  readonly Comment?: string;
}

interface IApiTransferDto {
  readonly Id: string;
  readonly Created: string;
  readonly Amount: { readonly currency: string; readonly value: number };
  readonly IsReturn?: boolean;
  readonly CreatorUserID?: string;
  readonly From?: IApiContactDto;
  readonly To?: IApiContactDto;
  readonly Due?: string;
  readonly Direction?: 'u2c' | 'c2u' | '3d-party';
  readonly IsOutstanding?: boolean;
  readonly Comment?: string;
}

interface IApiCreateTransferResponse {
  readonly Error?: string;
  readonly Transfer?: IApiTransferDto;
  readonly UserBalance?: Record<string, number>;
  readonly CounterpartyBalance?: Record<string, number>;
}

@Injectable()
export class DebtusService implements IDebtusService {
  private readonly sneatApiService = inject(SneatApiService);

  // ----- REAL endpoint: legacy thin create (kept, backwards compatible) -----
  public createDebtRecord(
    request: ICreateDebtRecordRequest,
  ): Observable<string> {
    return this.sneatApiService.post('debtus/create_debt_record', request);
  }

  // ===========================================================================
  // Fable: prototype demo data
  // No wired, authenticated Go HTTP endpoint exists yet for balances / contacts
  // / history (api4unsorted contacts CRUD is defined but not mounted in
  // backend/debtus/module.go; there is no balances endpoint). These read from
  // fixtures. SWAP POINT: replace each `of(...)` with a `sneatApiService.get`
  // once the endpoints are wired — the return types already match.
  // ===========================================================================

  public getContactBalances(spaceID: string): Observable<IContactBalance[]> {
    // SWAP: this.sneatApiService.get<IContactBalance[]>('api4debtus/user/contacts', new HttpParams().set('spaceID', spaceID))
    void spaceID;
    return of(DEMO_CONTACT_BALANCES.map((c) => ({ ...c })));
  }

  public getContactBalance(
    spaceID: string,
    contactID: string,
  ): Observable<IContactBalance> {
    void spaceID;
    const found =
      DEMO_CONTACT_BALANCES.find((c) => c.contactID === contactID) ??
      ({
        contactID,
        title: contactID,
        balance: {},
      } as IContactBalance);
    return of({ ...found });
  }

  public getTransfers(
    spaceID: string,
    contactID?: string,
  ): Observable<IDebtusTransfer[]> {
    // NOTE: GET /api4debtus/user/api4transfers exists but is currently broken
    // server-side (auth bug queries an empty userID). Using fixtures until the
    // backend fix lands. SWAP: this.sneatApiService.get('api4debtus/user/api4transfers', params).
    const all = demoTransfersForSpace(spaceID);
    return of(
      contactID
        ? all.filter((t) => t.counterpartyContactID === contactID)
        : all,
    );
  }

  // ===========================================================================
  // REAL endpoints below.
  // ===========================================================================

  /** Reads from demo fixtures; unknown ids error (no fabricated receipts). */
  public getTransfer(
    spaceID: string,
    transferID: string,
  ): Observable<IDebtusTransfer> {
    // The live GET transfer endpoint returns a perspective-resolved TransferDto.
    // For the prototype we resolve from fixtures so the receipt screen renders
    // without a live backend; SWAP to the real GET when running against a
    // deployed server:
    //   const params = new HttpParams().set('id', transferID);
    //   return this.sneatApiService
    //     .get<IApiTransferDto>('api4debtus/transfer', params)
    //     .pipe(map((dto) => this.mapTransferDto(dto, spaceID)));
    const found = demoTransfersForSpace(spaceID).find(
      (t) => t.id === transferID,
    );
    // Fable refactoring: a transfer that is not in the fixtures (i.e. any
    // REAL transfer just created via POST create-transfer) must be an error,
    // not a fabricated "Unknown / 0.00 USD / Outstanding" receipt — that was
    // presenting fiction as a financial record. The create/settle pages now
    // hand the created transfer to the details page via router state, so this
    // path is only hit on cold loads of unknown ids. The old synthesized
    // fallback is kept below (commented out) per the no-delete policy:
    //   return of(
    //     found ?? {
    //       id: transferID,
    //       direction: 'lend',
    //       amount: { currency: 'USD', value: 0 },
    //       counterpartyContactID: '',
    //       counterpartyTitle: 'Unknown',
    //       created: new Date().toISOString(),
    //       isReturn: false,
    //       isOutstanding: true,
    //       creatorSpaceID: spaceID,
    //     },
    //   );
    return found
      ? of(found)
      : throwError(
          () =>
            new Error(
              `Transfer "${transferID}" was not found (transfer reads are not wired to the live backend yet).`,
            ),
        );
  }

  /** REAL: POST /api4debtus/create-transfer (Firebase-authenticated). */
  public createTransfer(
    request: ICreateTransferRequest,
  ): Observable<ICreateTransferResponse> {
    const apiDirection = debtDirectionToApiDirection(request.direction);
    const body = {
      spaceID: request.spaceID,
      direction: apiDirection,
      amount: {
        currency: request.amount.currency,
        value: request.amount.value,
      },
      // For u2c (lend) the counterparty is the recipient (toContactID); for
      // c2u (borrow) the counterparty is the source (fromContactID).
      toContactID: apiDirection === 'u2c' ? request.contactID : undefined,
      fromContactID: apiDirection === 'c2u' ? request.contactID : undefined,
      // The backend CreateTransferRequest accepts both `note` and
      // `counterpartySpaceID` (facade4debtus/transfers_create_transfer_dto.go);
      // omitting them silently discarded the user's typed note and the
      // cross-space marker on a financial write.
      note: request.note,
      counterpartySpaceID: request.counterpartySpaceID || undefined,
      isReturn: request.isReturn ?? false,
      returnToTransferID: request.returnToTransferID,
      dueOn: request.dueOn,
    };
    return this.sneatApiService
      .post<IApiCreateTransferResponse>('api4debtus/create-transfer', body)
      .pipe(
        map((resp) => this.mapCreateResponse(resp, request)),
      );
  }

  /** Settle-up = a reverse-direction return transfer (mirrors the bot). */
  public settleUp(
    request: ISettleUpRequest,
  ): Observable<ICreateTransferResponse> {
    // Fable refactoring: the direction now comes from the request — the page
    // that shows the balance derives it via `settleDirectionForBalance` and is
    // the source of truth. Previously it was inferred from DEMO_CONTACT_BALANCES
    // fixtures, so any contact NOT in the fixtures got `'borrow'`
    // unconditionally and settling a debt the user owed recorded the WRONG
    // direction, increasing the imbalance. Old fixture-based inference kept
    // below per the no-delete policy:
    //   const contact = DEMO_CONTACT_BALANCES.find(
    //     (c) => c.contactID === request.contactID,
    //   );
    //   const currentValue = contact?.balance[request.amount.currency] ?? 0;
    //   const direction = settleDirectionForBalance(currentValue || 1);
    const direction = request.direction;
    return this.createTransfer({
      spaceID: request.spaceID,
      direction,
      amount: request.amount,
      contactID: request.contactID,
      contactTitle: request.contactTitle,
      isReturn: true,
      counterpartySpaceID: request.counterpartySpaceID,
    });
  }

  // ----- mapping helpers -----

  private mapCreateResponse(
    resp: IApiCreateTransferResponse,
    request: ICreateTransferRequest,
  ): ICreateTransferResponse {
    if (resp.Error) {
      throw new Error(resp.Error);
    }
    const transfer: IDebtusTransfer = resp.Transfer
      ? this.mapTransferDto(resp.Transfer, request.spaceID)
      : {
          // If the backend omits the transfer echo, synthesize from the request
          // so the UI can still navigate to a detail screen.
          id: `pending-${Date.now()}`,
          direction: request.direction,
          amount: request.amount,
          counterpartyContactID: request.contactID,
          counterpartyTitle: request.contactTitle ?? request.contactID,
          note: request.note,
          created: new Date().toISOString(),
          dueOn: request.dueOn,
          isReturn: request.isReturn ?? false,
          isOutstanding: true,
          creatorSpaceID: request.spaceID,
          counterpartySpaceID: request.counterpartySpaceID,
        };
    return {
      transfer,
      userBalance: (resp.UserBalance ?? {}) as ICreateTransferResponse['userBalance'],
      counterpartyBalance: (resp.CounterpartyBalance ??
        {}) as ICreateTransferResponse['counterpartyBalance'],
    };
  }

  private mapTransferDto(
    dto: IApiTransferDto,
    spaceID: string,
  ): IDebtusTransfer {
    const counterparty = dto.To ?? dto.From;
    const direction = dto.Direction
      ? apiDirectionToDebtDirection(dto.Direction)
      : 'lend';
    return {
      id: dto.Id,
      direction,
      amount: {
        currency: (dto.Amount?.currency ?? 'USD') as
          | 'USD'
          | 'EUR',
        value: dto.Amount?.value ?? 0,
      },
      counterpartyContactID: counterparty?.ID ?? '',
      counterpartyTitle: counterparty?.Name ?? 'Unknown',
      note: dto.Comment,
      created: dto.Created ?? new Date().toISOString(),
      dueOn: dto.Due,
      isReturn: dto.IsReturn ?? false,
      isOutstanding: dto.IsOutstanding ?? true,
      creatorSpaceID: spaceID,
    };
  }
}
