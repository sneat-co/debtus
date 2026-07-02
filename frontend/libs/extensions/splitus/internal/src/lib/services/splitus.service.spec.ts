import { HttpParams } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';
import { SneatApiService } from '@sneat/api';
import {
  ICreateSplitRequest,
  ISplitShare,
} from '@sneat/extension-splitus-contract';
import { of } from 'rxjs';
import { SplitusService } from './splitus.service';

describe('SplitusService', () => {
  let post: ReturnType<typeof vi.fn>;
  let get: ReturnType<typeof vi.fn>;
  let service: SplitusService;

  beforeEach(() => {
    post = vi.fn();
    get = vi.fn();
    TestBed.configureTestingModule({
      providers: [
        SplitusService,
        { provide: SneatApiService, useValue: { post, get } },
      ],
    });
    service = TestBed.inject(SplitusService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('createSplit', () => {
    it('posts the request body verbatim to api4splitus/create-split', () => {
      post.mockReturnValue(of({ id: 'bill1', transfers: [] }));
      const request: ICreateSplitRequest = {
        spaceID: 'space1',
        title: 'Dinner',
        currency: 'EUR',
        amount: '90.00',
        participantContactIDs: ['cBea', 'cCam'],
      };

      service.createSplit(request).subscribe();

      expect(post).toHaveBeenCalledWith('api4splitus/create-split', {
        spaceID: 'space1',
        title: 'Dinner',
        currency: 'EUR',
        amount: '90.00',
        splitMode: undefined,
        participantContactIDs: ['cBea', 'cCam'],
        shares: undefined,
      });
    });

    it('forwards splitMode and shares for a custom split', () => {
      post.mockReturnValue(of({ id: 'bill1', transfers: [] }));
      const shares: ISplitShare[] = [
        { contactID: '', amount: '30.00' },
        { contactID: 'cBea', amount: '60.00' },
      ];
      const request: ICreateSplitRequest = {
        spaceID: 'space1',
        currency: 'EUR',
        amount: '90.00',
        splitMode: 'exact-amount',
        participantContactIDs: ['cBea'],
        shares,
      };

      service.createSplit(request).subscribe();

      expect(post).toHaveBeenCalledWith('api4splitus/create-split', {
        spaceID: 'space1',
        title: undefined,
        currency: 'EUR',
        amount: '90.00',
        splitMode: 'exact-amount',
        participantContactIDs: ['cBea'],
        shares,
      });
    });

    it('maps the created bill id and transfers, converting cents to major units', () => {
      post.mockReturnValue(
        of({
          id: 'bill1',
          transfers: [
            { id: 'transfer1', contactID: 'cBea', amount: 3000 },
            { id: 'transfer2', contactID: 'cCam', amount: 3000 },
          ],
        }),
      );

      let result: unknown;
      service
        .createSplit({
          spaceID: 'space1',
          currency: 'EUR',
          amount: '90.00',
          participantContactIDs: ['cBea', 'cCam'],
        })
        .subscribe((r) => (result = r));

      expect(result).toEqual({
        id: 'bill1',
        transfers: [
          { id: 'transfer1', contactID: 'cBea', amount: 30 },
          { id: 'transfer2', contactID: 'cCam', amount: 30 },
        ],
      });
    });
  });

  describe('getSplit', () => {
    it('requests api4splitus/split with spaceID and id query params', () => {
      get.mockReturnValue(
        of({
          id: 'bill1',
          title: 'Dinner',
          currency: 'EUR',
          amount: 9000,
          status: 'outstanding',
          participants: [],
        }),
      );

      service.getSplit('space1', 'bill1').subscribe();

      expect(get).toHaveBeenCalledWith(
        'api4splitus/split',
        new HttpParams().set('spaceID', 'space1').set('id', 'bill1'),
      );
    });

    it('maps each participant status verbatim from the API — no client-side settled computation', () => {
      get.mockReturnValue(
        of({
          id: 'bill1',
          title: 'Dinner',
          currency: 'EUR',
          amount: 9000,
          status: 'outstanding',
          participants: [
            {
              userID: 'alex',
              name: 'Alex',
              share: 3000,
              isPayer: true,
              status: 'settled',
            },
            {
              contactID: 'cBea',
              name: 'Bea',
              share: 3000,
              status: 'settled',
            },
            {
              contactID: 'cCam',
              name: 'Cam',
              share: 3000,
              status: 'outstanding',
            },
          ],
        }),
      );

      let result: unknown;
      service.getSplit('space1', 'bill1').subscribe((r) => (result = r));

      expect(result).toEqual({
        id: 'bill1',
        title: 'Dinner',
        currency: 'EUR',
        amount: 90,
        status: 'outstanding',
        participants: [
          {
            contactID: undefined,
            userID: 'alex',
            name: 'Alex',
            share: 30,
            isPayer: true,
            status: 'settled',
          },
          {
            contactID: 'cBea',
            userID: undefined,
            name: 'Bea',
            share: 30,
            isPayer: undefined,
            status: 'settled',
          },
          {
            contactID: 'cCam',
            userID: undefined,
            name: 'Cam',
            share: 30,
            isPayer: undefined,
            status: 'outstanding',
          },
        ],
      });
    });
  });

  describe('getSplits', () => {
    it('requests api4splitus/splits with the spaceID query param and maps the list', () => {
      get.mockReturnValue(
        of({
          splits: [
            {
              id: 'bill1',
              title: 'Dinner',
              amount: 9000,
              currency: 'EUR',
              status: 'outstanding',
              membersCount: 3,
            },
            {
              id: 'bill2',
              amount: 5000,
              currency: 'USD',
              status: 'settled',
              membersCount: 2,
            },
          ],
        }),
      );

      let result: unknown;
      service.getSplits('space1').subscribe((r) => (result = r));

      expect(get).toHaveBeenCalledWith(
        'api4splitus/splits',
        new HttpParams().set('spaceID', 'space1'),
      );
      expect(result).toEqual([
        {
          id: 'bill1',
          title: 'Dinner',
          amount: 90,
          currency: 'EUR',
          status: 'outstanding',
          membersCount: 3,
        },
        {
          id: 'bill2',
          title: undefined,
          amount: 50,
          currency: 'USD',
          status: 'settled',
          membersCount: 2,
        },
      ]);
    });
  });
});
