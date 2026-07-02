import { TestBed } from '@angular/core/testing';
import { SneatApiService } from '@sneat/api';
import {
  ICreateTransferRequest,
  ISettleUpRequest,
} from '@sneat/extension-debtus-contract';
import { firstValueFrom, of } from 'rxjs';
import { DebtusService } from './debtus.service';

describe('DebtusService', () => {
  let post: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    post = vi.fn().mockReturnValue(of({}));
    TestBed.configureTestingModule({
      providers: [DebtusService, { provide: SneatApiService, useValue: { post } }],
    });
  });

  it('should be created', () => {
    expect(TestBed.inject(DebtusService)).toBeTruthy();
  });

  describe('createTransfer', () => {
    it('sends note and counterpartySpaceID to the backend', async () => {
      const service = TestBed.inject(DebtusService);
      const request: ICreateTransferRequest = {
        spaceID: 'space1',
        direction: 'lend',
        amount: { currency: 'EUR', value: 12.5 },
        contactID: 'c1',
        contactTitle: 'Alice',
        note: 'lunch money',
        counterpartySpaceID: 'space2',
      };
      await firstValueFrom(service.createTransfer(request));
      expect(post).toHaveBeenCalledWith(
        'api4debtus/create-transfer',
        expect.objectContaining({
          note: 'lunch money',
          counterpartySpaceID: 'space2',
          toContactID: 'c1',
        }),
      );
    });
  });

  describe('getTransfer', () => {
    it('errors for ids not in the fixtures instead of fabricating a receipt', async () => {
      const service = TestBed.inject(DebtusService);
      await expect(
        firstValueFrom(service.getTransfer('space1', 'no-such-transfer')),
      ).rejects.toThrow(/not found/);
    });
  });

  describe('settleUp', () => {
    it('uses the direction from the request, not fixture-derived', async () => {
      const service = TestBed.inject(DebtusService);
      const request: ISettleUpRequest = {
        spaceID: 'space1',
        contactID: 'not-in-fixtures',
        amount: { currency: 'EUR', value: 5 },
        // The user owes this contact, so settling means the user gives money
        // back => 'lend' (u2c). Fixture-based inference would have said
        // 'borrow' (c2u) for any contact missing from the fixtures.
        direction: 'lend',
      };
      await firstValueFrom(service.settleUp(request));
      expect(post).toHaveBeenCalledWith(
        'api4debtus/create-transfer',
        expect.objectContaining({
          direction: 'u2c',
          isReturn: true,
          toContactID: 'not-in-fixtures',
        }),
      );
    });
  });
});
