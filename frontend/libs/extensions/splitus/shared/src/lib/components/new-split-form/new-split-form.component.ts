import {
  Component,
  DestroyRef,
  Input,
  computed,
  inject,
  signal,
} from '@angular/core';
import { takeUntilDestroyed, toSignal } from '@angular/core/rxjs-interop';
import {
  FormControl,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import {
  IonButton,
  IonCard,
  IonCardHeader,
  IonCardTitle,
  IonCheckbox,
  IonInput,
  IonItem,
  IonLabel,
  IonNote,
  IonSegment,
  IonSegmentButton,
  IonSelect,
  IonSelectOption,
  IonSpinner,
} from '@ionic/angular/standalone';
import { ErrorLogger, IErrorLogger, IIdAndBrief } from '@sneat/core';
import {
  CONTACTUS_SPACE_SERVICE,
  IContactBrief,
} from '@sneat/extension-contactus-contract';
import {
  CurrencyCode,
  ICreateSplitRequest,
  ISplitShare,
  SPLITUS_SERVICE,
  SplitMode,
} from '@sneat/extension-splitus-contract';
import { ISpaceContext } from '@sneat/space-models';
import { SpaceNavService } from '@sneat/space-services';
import { Subscription } from 'rxjs';
import {
  formatHundredths,
  formatPercentValue,
  parseDecimal,
  reconcileShares,
} from './split-shares';

interface IParticipantChoice {
  readonly id: string;
  readonly title: string;
}

/** One row of the custom-share editor. `id === ''` is the payer. */
interface IShareRow {
  readonly id: string;
  readonly title: string;
}

/** Pulls a human message out of an HTTP error (Go handlers reply plain text). */
function extractServerMessage(err: unknown): string | undefined {
  if (typeof err !== 'object' || !err) {
    return typeof err === 'string' && err.trim() ? err.trim() : undefined;
  }
  const e = err as { error?: unknown; message?: unknown };
  if (typeof e.error === 'string' && e.error.trim()) {
    return e.error.trim();
  }
  if (typeof e.error === 'object' && e.error) {
    const m = (e.error as { message?: unknown }).message;
    if (typeof m === 'string' && m.trim()) {
      return m.trim();
    }
  }
  return typeof e.message === 'string' && e.message.trim()
    ? e.message.trim()
    : undefined;
}

// Create-split form: amount + currency, participants picked from the space's
// existing contactus people (no free-text name entry — AC
// participants-from-contactus-membership-enforced), and an
// equally/exact-amount/percentage share editor with live reconciliation that
// blocks submit until custom shares sum to the total (AC
// custom-shares-must-sum). On success redirects to the new split's details
// page with replaceUrl (flows.md create→details default).
@Component({
  selector: 'sneat-splitus-new-split-form',
  templateUrl: './new-split-form.component.html',
  imports: [
    ReactiveFormsModule,
    IonCard,
    IonCardHeader,
    IonCardTitle,
    IonItem,
    IonLabel,
    IonNote,
    IonSelect,
    IonSelectOption,
    IonInput,
    IonCheckbox,
    IonSegment,
    IonSegmentButton,
    IonButton,
    IonSpinner,
  ],
})
export class NewSplitFormComponent {
  private readonly errorLogger = inject<IErrorLogger>(ErrorLogger);
  private readonly splitusService = inject(SPLITUS_SERVICE);
  private readonly contactusSpaceService = inject(CONTACTUS_SPACE_SERVICE);
  private readonly spaceNavService = inject(SpaceNavService);
  private readonly destroyRef = inject(DestroyRef);

  protected readonly $space = signal<ISpaceContext | undefined>(undefined);
  @Input({ required: true }) public set space(value: ISpaceContext | undefined) {
    this.$space.set(value);
    this.watchContacts(value?.id);
  }

  protected readonly $currentUserID = signal<string | undefined>(undefined);
  /** The authenticated user's ID — used to exclude the payer's own contact. */
  @Input() public set currentUserID(value: string | undefined) {
    this.$currentUserID.set(value);
  }

  // ---- participants (sourced from contactus — never typed in) ----

  private contactsSubscription?: Subscription;
  private watchedSpaceID?: string;
  protected readonly $contacts = signal<
    IIdAndBrief<IContactBrief>[] | undefined
  >(undefined);
  protected readonly $contactsError = signal<string | undefined>(undefined);

  private watchContacts(spaceID: string | undefined): void {
    if (spaceID === this.watchedSpaceID) {
      return;
    }
    this.watchedSpaceID = spaceID;
    this.contactsSubscription?.unsubscribe();
    this.$contacts.set(undefined);
    this.$contactsError.set(undefined);
    if (!spaceID) {
      return;
    }
    this.contactsSubscription = this.contactusSpaceService
      .watchContactBriefs(spaceID)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (contacts) => this.$contacts.set(contacts),
        error: (err) => {
          this.$contactsError.set('Failed to load contacts.');
          this.errorLogger.logError(err, 'Failed to load space contacts', {
            show: false,
          });
        },
      });
  }

  /**
   * The space's contactus people offered as participants. The payer (the
   * authenticated user) is always a participant and must not be listed in
   * `participantContactIDs`, so their own contact is excluded from the picker.
   */
  protected readonly $participantChoices = computed<
    readonly IParticipantChoice[] | undefined
  >(() => {
    const userID = this.$currentUserID();
    return this.$contacts()
      ?.filter((c) => !userID || c.brief?.userID !== userID)
      .map((c) => ({
        id: c.id,
        title: c.brief?.title || c.brief?.shortTitle || c.id,
      }));
  });

  protected readonly $selectedContactIDs = signal<readonly string[]>([]);

  protected onParticipantToggle(contactID: string, checked: boolean): void {
    this.$selectedContactIDs.update((ids) => {
      const without = ids.filter((id) => id !== contactID);
      return checked ? [...without, contactID] : without;
    });
  }

  protected onParticipantChecked(contactID: string, event: Event): void {
    const checked = !!(event as CustomEvent<{ checked?: boolean }>).detail
      ?.checked;
    this.onParticipantToggle(contactID, checked);
  }

  // ---- expense fields ----

  protected readonly title = new FormControl<string>('', { nonNullable: true });
  // TODO: pre-fill from a per-space default currency once spaces carry one;
  // EUR is the app-wide default (same as debtus's new-transfer form).
  protected readonly currency = new FormControl<CurrencyCode>('EUR', {
    nonNullable: true,
  });
  protected readonly amount = new FormControl<number | null>(null, [
    Validators.required,
    Validators.min(0.01),
  ]);
  private readonly $amount = toSignal(this.amount.valueChanges, {
    initialValue: this.amount.value,
  });

  protected readonly currencies: CurrencyCode[] = ['EUR', 'USD'];

  // ---- split mode + custom shares ----

  protected readonly $splitMode = signal<SplitMode>('equally');

  protected setSplitMode(mode: SplitMode): void {
    this.$splitMode.set(mode);
  }

  protected onSplitModeChange(event: Event): void {
    const value = (event as CustomEvent<{ value?: string }>).detail?.value;
    if (
      value === 'equally' ||
      value === 'exact-amount' ||
      value === 'percentage'
    ) {
      this.setSplitMode(value);
    }
  }

  /** Raw share inputs keyed by contact ID ('' = the payer). */
  protected readonly $shareValues = signal<Readonly<Record<string, string>>>(
    {},
  );

  protected setShareValue(contactID: string, value: string): void {
    this.$shareValues.update((values) => ({ ...values, [contactID]: value }));
  }

  protected shareValue(contactID: string): string {
    const value: string | undefined = this.$shareValues()[contactID];
    return value ?? '';
  }

  protected onShareInput(contactID: string, event: Event): void {
    const value = (event as CustomEvent<{ value?: string | null }>).detail
      ?.value;
    this.setShareValue(contactID, value ?? '');
  }

  /** Share-editor rows: the payer first, then each selected participant. */
  protected readonly $shareRows = computed<readonly IShareRow[]>(() => {
    const choices = this.$participantChoices() ?? [];
    const selected = this.$selectedContactIDs();
    return [
      { id: '', title: 'You (payer)' },
      ...choices.filter((c) => selected.includes(c.id)),
    ];
  });

  /** Live reconciliation of custom shares against the expense total. */
  protected readonly $reconciliation = computed(() =>
    reconcileShares(
      this.$splitMode(),
      parseDecimal(this.$amount()),
      this.$shareRows().map((row) => this.$shareValues()[row.id]),
    ),
  );

  protected readonly $canSubmit = computed(() => {
    const total = parseDecimal(this.$amount());
    return (
      total !== undefined &&
      total > 0 &&
      this.$selectedContactIDs().length > 0 &&
      this.$reconciliation().ok
    );
  });

  // ---- submit + exit ----

  protected readonly $submitting = signal(false);
  protected readonly $submitError = signal<string | undefined>(undefined);

  protected submit(): void {
    this.amount.markAsTouched();
    if (this.$submitting() || !this.$canSubmit()) {
      return;
    }
    const space = this.$space();
    if (!space?.id) {
      return;
    }
    const totalCents = parseDecimal(this.$amount());
    if (totalCents === undefined) {
      return;
    }
    const mode = this.$splitMode();
    const shares: ISplitShare[] | undefined =
      mode === 'equally'
        ? undefined
        : this.$shareRows().map((row) => {
            const value = parseDecimal(this.$shareValues()[row.id]) ?? 0;
            return mode === 'percentage'
              ? { contactID: row.id, percent: formatPercentValue(value) }
              : { contactID: row.id, amount: formatHundredths(value) };
          });
    const request: ICreateSplitRequest = {
      spaceID: space.id,
      title: this.title.value.trim() || undefined,
      currency: this.currency.value,
      amount: formatHundredths(totalCents),
      splitMode: mode,
      participantContactIDs: [...this.$selectedContactIDs()],
      shares,
    };
    this.$submitting.set(true);
    this.$submitError.set(undefined);
    this.splitusService.createSplit(request).subscribe({
      next: (response) => {
        // flows.md: after a successful create, go to the new entity's details,
        // replacing the create page in history so Back skips the stale form.
        this.spaceNavService.navigateForwardToSpacePage(
          space,
          `split/${response.id}`,
          { replaceUrl: true },
        );
      },
      error: (err) => {
        this.$submitting.set(false);
        const serverMessage = extractServerMessage(err);
        this.$submitError.set(
          serverMessage
            ? `Failed to create split: ${serverMessage}`
            : 'Failed to create split. Please try again.',
        );
        this.errorLogger.logError(err, 'Failed to create split', {
          show: false,
        });
      },
    });
  }
}
