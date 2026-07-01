import { Component, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import {
  FormControl,
  FormGroup,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import {
  IonBackButton,
  IonButton,
  IonButtons,
  IonCard,
  IonCardContent,
  IonCardHeader,
  IonCardTitle,
  IonContent,
  IonHeader,
  IonInput,
  IonItem,
  IonLabel,
  IonNote,
  IonSelect,
  IonSelectOption,
  IonSpinner,
  IonTitle,
  IonToolbar,
  ToastController,
} from '@ionic/angular/standalone';
import {
  CurrencyCode,
  DEBTUS_SERVICE,
  IContactBalance,
  ISettleUpRequest,
  formatSignedBalance,
  round2,
} from '@sneat/extension-debtus-contract';
import {
  SpaceComponentBaseParams,
  SpacePageBaseComponent,
} from '@sneat/space-components';
import { ClassName } from '@sneat/ui';
import { combineLatest, switchMap } from 'rxjs';

// Settle up — records a settling (return) transfer against a counterparty
// balance. Mirrors the bot's "Returned fully/partially" flow: the backend nets
// the return against outstanding transfers.
@Component({
  selector: 'sneat-debtus-settle-up-page',
  templateUrl: './settle-up-page.component.html',
  imports: [
    ReactiveFormsModule,
    IonHeader,
    IonToolbar,
    IonButtons,
    IonBackButton,
    IonTitle,
    IonContent,
    IonCard,
    IonCardHeader,
    IonCardTitle,
    IonCardContent,
    IonItem,
    IonLabel,
    IonInput,
    IonSelect,
    IonSelectOption,
    IonButton,
    IonNote,
    IonSpinner,
  ],
  providers: [
    { provide: ClassName, useValue: 'SettleUpPageComponent' },
    SpaceComponentBaseParams,
  ],
})
export class SettleUpPageComponent extends SpacePageBaseComponent {
  private readonly debtusService = inject(DEBTUS_SERVICE);
  private readonly toastController = inject(ToastController);

  protected readonly $loading = signal(true);
  protected readonly $submitting = signal(false);
  protected readonly $error = signal<string | undefined>(undefined);
  protected readonly $contact = signal<IContactBalance | undefined>(undefined);
  protected contactID = '';

  protected readonly formatSignedBalance = formatSignedBalance;

  protected readonly currency = new FormControl<CurrencyCode>('EUR', {
    nonNullable: true,
  });
  protected readonly amount = new FormControl<number | null>(null, [
    Validators.required,
    Validators.min(0.01),
  ]);
  protected readonly form = new FormGroup({
    currency: this.currency,
    amount: this.amount,
  });

  protected readonly currencies: CurrencyCode[] = ['EUR', 'USD'];

  constructor() {
    super();
    combineLatest([this.spaceIDChanged$, this.route.queryParamMap])
      .pipe(
        switchMap(([spaceID, params]) => {
          this.contactID = params.get('contactID') ?? '';
          // Back should return to the contact detail this settle-up was
          // opened from, not a fixed page.
          this.$defaultBackUrlSpacePath.set(
            `debtus-contact/${this.contactID}`,
          );
          this.$loading.set(true);
          this.$error.set(undefined);
          return this.debtusService.getContactBalance(
            spaceID ?? '',
            this.contactID,
          );
        }),
        takeUntilDestroyed(),
      )
      .subscribe({
        next: (contact) => {
          this.$contact.set(contact);
          // Default the settle amount/currency to the outstanding balance.
          const entries = Object.entries(contact.balance).filter(
            ([, v]) => v && round2(v) !== 0,
          );
          if (entries.length) {
            const [cur, val] = entries[0];
            this.currency.setValue(cur as CurrencyCode);
            this.amount.setValue(Math.abs(round2(val as number)));
          }
          this.$loading.set(false);
        },
        error: (err) => {
          this.$loading.set(false);
          this.$error.set('Failed to load contact');
          this.errorLogger.logError(err, 'Failed to load contact for settle');
        },
      });
  }

  protected submit(): void {
    this.form.markAllAsTouched();
    const amount = this.amount.value;
    if (!amount || amount <= 0) {
      return;
    }
    const spaceID = this.$spaceID();
    if (!spaceID || !this.contactID) {
      return;
    }
    // Mirrors the bot's "Is it returned in full?" confirmation step
    // (askIfReturnedInFull in transfer_return.go) before recording the
    // settling transfer — this is a financial record, not a reversible
    // toggle, so it needs an explicit yes.
    const title = this.$contact()?.title ?? 'this contact';
    if (
      !confirm(
        `Record settlement of ${this.currency.value} ${amount.toFixed(2)} with ${title}?`,
      )
    ) {
      return;
    }
    const request: ISettleUpRequest = {
      spaceID,
      contactID: this.contactID,
      contactTitle: this.$contact()?.title,
      amount: { currency: this.currency.value, value: amount },
      counterpartySpaceID: this.$contact()?.counterpartySpaceID,
    };
    this.$submitting.set(true);
    this.debtusService.settleUp(request).subscribe({
      next: (resp) => {
        this.$submitting.set(false);
        this.spaceNav.navigateForwardToSpacePage(
          this.space,
          `transfer/${resp.transfer.id}`,
          { replaceUrl: true },
        );
      },
      error: (err) => {
        this.$submitting.set(false);
        this.errorLogger.logError(err, 'Failed to settle up');
        this.showToast('Failed to settle up. Please try again.');
      },
    });
  }

  private async showToast(message: string): Promise<void> {
    const toast = await this.toastController.create({
      message,
      duration: 3000,
      color: 'danger',
    });
    await toast.present();
  }
}
