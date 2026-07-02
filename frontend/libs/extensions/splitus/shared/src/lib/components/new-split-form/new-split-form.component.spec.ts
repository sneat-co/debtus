import { CUSTOM_ELEMENTS_SCHEMA } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ErrorLogger, IIdAndBrief } from '@sneat/core';
import {
  CONTACTUS_SPACE_SERVICE,
  IContactBrief,
} from '@sneat/extension-contactus-contract';
import { SPLITUS_SERVICE } from '@sneat/extension-splitus-contract';
import { ISpaceContext } from '@sneat/space-models';
import { SpaceNavService } from '@sneat/space-services';
import { Observable, Subject, of, throwError } from 'rxjs';
import { NewSplitFormComponent } from './new-split-form.component';

const space: ISpaceContext = { id: 'space1', type: 'family' };

const brief = (
  id: string,
  title: string,
  userID?: string,
): IIdAndBrief<IContactBrief> => ({
  id,
  brief: { type: 'person', title, userID },
});

// Space contacts as contactus returns them: the payer (Alex, the current
// user) plus two plain contacts, Bea and Cam.
const contactBriefs = [
  brief('alex', 'Alex', 'user-alex'),
  brief('bea', 'Bea'),
  brief('cam', 'Cam'),
];

describe('NewSplitFormComponent', () => {
  let fixture: ComponentFixture<NewSplitFormComponent>;
  let component: NewSplitFormComponent;

  const createSplit = vi.fn();
  const watchContactBriefs = vi.fn();
  const navigateForwardToSpacePage = vi.fn();

  const setup = (
    briefs$: Observable<IIdAndBrief<IContactBrief>[]> = of(contactBriefs),
  ) => {
    createSplit.mockReset();
    watchContactBriefs.mockReset().mockReturnValue(briefs$);
    navigateForwardToSpacePage.mockReset().mockResolvedValue(true);

    TestBed.configureTestingModule({
      imports: [NewSplitFormComponent],
      schemas: [CUSTOM_ELEMENTS_SCHEMA],
      providers: [
        {
          provide: ErrorLogger,
          useValue: { logError: vi.fn(), logErrorHandler: () => vi.fn() },
        },
        {
          provide: SPLITUS_SERVICE,
          useValue: { createSplit, getSplit: vi.fn(), getSplits: vi.fn() },
        },
        {
          provide: CONTACTUS_SPACE_SERVICE,
          useValue: { watchContactBriefs },
        },
        {
          provide: SpaceNavService,
          useValue: { navigateForwardToSpacePage },
        },
      ],
    }).overrideComponent(NewSplitFormComponent, {
      set: {
        imports: [],
        providers: [],
        schemas: [CUSTOM_ELEMENTS_SCHEMA],
      },
    });

    fixture = TestBed.createComponent(NewSplitFormComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('space', space);
    fixture.componentRef.setInput('currentUserID', 'user-alex');
  };

  // Reach protected members without widening the component's public API.
  /* eslint-disable @typescript-eslint/no-explicit-any */
  const c = () => component as any;
  /* eslint-enable @typescript-eslint/no-explicit-any */

  const enterAmount = (value: number) => c().amount.setValue(value);
  const selectParticipants = (...ids: string[]) =>
    ids.forEach((id) => c().onParticipantToggle(id, true));

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('should create', () => {
    setup();
    expect(component).toBeTruthy();
  });

  describe('participant picker (splitus#ac:participants-from-contactus-membership-enforced)', () => {
    it("offers the space's existing contactus people — no re-entry of names", () => {
      setup();
      expect(watchContactBriefs).toHaveBeenCalledWith('space1');
      const choices = c().$participantChoices();
      expect(choices?.map((p: { title: string }) => p.title)).toEqual(
        expect.arrayContaining(['Bea', 'Cam']),
      );
    });

    it("excludes the payer's own contact — the payer is always a participant", () => {
      setup();
      const ids = c()
        .$participantChoices()
        ?.map((p: { id: string }) => p.id);
      expect(ids).toEqual(['bea', 'cam']);
    });

    it('blocks submit until at least one participant is selected', () => {
      setup();
      enterAmount(100);
      expect(c().$canSubmit()).toBe(false);
      selectParticipants('bea');
      expect(c().$canSubmit()).toBe(true);
    });

    it('sends only selected contactus contact IDs, scoped to the space', () => {
      setup();
      createSplit.mockReturnValue(of({ id: 'split1', transfers: [] }));
      enterAmount(90);
      selectParticipants('bea', 'cam');
      c().submit();
      expect(createSplit).toHaveBeenCalledWith(
        expect.objectContaining({
          spaceID: 'space1',
          participantContactIDs: ['bea', 'cam'],
        }),
      );
    });
  });

  describe('equal split (default)', () => {
    it('sends splitMode "equally" without shares', () => {
      setup();
      createSplit.mockReturnValue(of({ id: 'split1', transfers: [] }));
      enterAmount(90);
      selectParticipants('bea', 'cam');
      c().submit();
      expect(createSplit).toHaveBeenCalledWith(
        expect.objectContaining({
          splitMode: 'equally',
          amount: '90.00',
          currency: 'EUR',
          shares: undefined,
        }),
      );
    });
  });

  describe('custom shares reconciliation (splitus#ac:custom-shares-must-sum)', () => {
    it('blocks submit with an explanatory delta message while exact amounts do not sum to the total', () => {
      setup();
      enterAmount(100);
      selectParticipants('bea', 'cam');
      c().setSplitMode('exact-amount');
      c().setShareValue('', '50'); // payer
      c().setShareValue('bea', '30');
      c().setShareValue('cam', '10');

      expect(c().$reconciliation().ok).toBe(false);
      expect(c().$reconciliation().error).toContain('90.00');
      expect(c().$reconciliation().error).toContain('short');
      expect(c().$canSubmit()).toBe(false);

      c().submit();
      expect(createSplit).not.toHaveBeenCalled();
    });

    it('reconciles live as shares change and then allows submit', () => {
      setup();
      createSplit.mockReturnValue(of({ id: 'split1', transfers: [] }));
      enterAmount(100);
      selectParticipants('bea', 'cam');
      c().setSplitMode('exact-amount');
      c().setShareValue('', '50');
      c().setShareValue('bea', '30');
      c().setShareValue('cam', '10');
      expect(c().$canSubmit()).toBe(false);

      c().setShareValue('cam', '20'); // now 50 + 30 + 20 == 100
      expect(c().$reconciliation()).toEqual({ ok: true });
      expect(c().$canSubmit()).toBe(true);

      c().submit();
      expect(createSplit).toHaveBeenCalledWith(
        expect.objectContaining({
          splitMode: 'exact-amount',
          amount: '100.00',
          shares: [
            { contactID: '', amount: '50.00' },
            { contactID: 'bea', amount: '30.00' },
            { contactID: 'cam', amount: '20.00' },
          ],
        }),
      );
    });

    it('blocks submit until percentages sum to 100%', () => {
      setup();
      createSplit.mockReturnValue(of({ id: 'split1', transfers: [] }));
      enterAmount(100);
      selectParticipants('bea', 'cam');
      c().setSplitMode('percentage');
      c().setShareValue('', '33.33');
      c().setShareValue('bea', '33.33');
      c().setShareValue('cam', '33.33');

      expect(c().$reconciliation().ok).toBe(false);
      expect(c().$reconciliation().error).toContain('100%');
      expect(c().$canSubmit()).toBe(false);
      c().submit();
      expect(createSplit).not.toHaveBeenCalled();

      c().setShareValue('cam', '33.34');
      expect(c().$canSubmit()).toBe(true);
      c().submit();
      expect(createSplit).toHaveBeenCalledWith(
        expect.objectContaining({
          splitMode: 'percentage',
          shares: [
            { contactID: '', percent: '33.33' },
            { contactID: 'bea', percent: '33.33' },
            { contactID: 'cam', percent: '33.34' },
          ],
        }),
      );
    });

    it('surfaces the server rejection message and stays on the form', () => {
      setup();
      createSplit.mockReturnValue(
        throwError(() => ({
          status: 400,
          error: 'shares must sum up to the total amount',
        })),
      );
      enterAmount(100);
      selectParticipants('bea');
      c().setSplitMode('exact-amount');
      c().setShareValue('', '50');
      c().setShareValue('bea', '50');

      c().submit();

      expect(c().$submitError()).toContain(
        'shares must sum up to the total amount',
      );
      expect(c().$submitting()).toBe(false);
      expect(navigateForwardToSpacePage).not.toHaveBeenCalled();
    });
  });

  describe('exit — redirect to the new split details (flows.md)', () => {
    it('navigates to split/:id with replaceUrl on success', () => {
      setup();
      createSplit.mockReturnValue(of({ id: 'split123', transfers: [] }));
      enterAmount(90);
      selectParticipants('bea');
      c().submit();
      expect(navigateForwardToSpacePage).toHaveBeenCalledWith(
        space,
        'split/split123',
        { replaceUrl: true },
      );
    });

    it('blocks double-submit while the create call is in flight', () => {
      setup();
      const pending$ = new Subject<{ id: string; transfers: [] }>();
      createSplit.mockReturnValue(pending$);
      enterAmount(90);
      selectParticipants('bea');
      c().submit();
      expect(c().$submitting()).toBe(true);
      c().submit();
      expect(createSplit).toHaveBeenCalledTimes(1);
    });
  });
});
