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
  IonContent,
  IonHeader,
  IonInput,
  IonItem,
  IonLabel,
  IonNote,
  IonSegment,
  IonSegmentButton,
  IonSelect,
  IonSelectOption,
  IonSpinner,
  IonTextarea,
  IonTitle,
  IonToolbar,
  ToastController,
} from '@ionic/angular/standalone';
import { IContactContext } from '@sneat/extension-contactus-contract';
import { ContactInputComponent } from '@sneat/extension-contactus-shared';
import {
  CurrencyCode,
  DEBTUS_SERVICE,
  DebtDirection,
  ICreateTransferRequest,
} from '@sneat/extension-debtus-contract';
import {
  SpaceComponentBaseParams,
  SpacePageBaseComponent,
} from '@sneat/space-components';
import { ClassName } from '@sneat/ui';
import { first } from 'rxjs';

// Create transfer — the primary write flow. Mirrors the bot's lend/borrow
// wizard: direction, counterparty (a contactus space contact), amount +
// currency, optional note. Reuses the contactus `sneat-contact-input` picker
// so the counterparty is a real space contact, not a debtus-private person.
@Component({
  selector: 'sneat-debtus-new-transfer-page',
  templateUrl: './new-transfer-page.component.html',
  imports: [
    ReactiveFormsModule,
    IonHeader,
    IonToolbar,
    IonButtons,
    IonBackButton,
    IonTitle,
    IonContent,
    IonCard,
    IonCardContent,
    IonItem,
    IonLabel,
    IonInput,
    IonTextarea,
    IonSelect,
    IonSelectOption,
    IonSegment,
    IonSegmentButton,
    IonButton,
    IonNote,
    IonSpinner,
    ContactInputComponent,
  ],
  providers: [
    { provide: ClassName, useValue: 'NewTransferPageComponent' },
    SpaceComponentBaseParams,
  ],
})
export class NewTransferPageComponent extends SpacePageBaseComponent {
  private readonly debtusService = inject(DEBTUS_SERVICE);
  private readonly toastController = inject(ToastController);

  protected readonly $submitting = signal(false);
  protected readonly $pickedContact = signal<IContactContext | undefined>(
    undefined,
  );
  /** Set when arriving from a contact detail page (?contactID=…). */
  protected prefilledContactID?: string;

  protected readonly direction = new FormControl<DebtDirection>('lend', {
    nonNullable: true,
  });
  protected readonly currency = new FormControl<CurrencyCode>('EUR', {
    nonNullable: true,
  });
  protected readonly amount = new FormControl<number | null>(null, [
    Validators.required,
    Validators.min(0.01),
  ]);
  protected readonly counterpartyName = new FormControl<string>('', {
    nonNullable: true,
  });
  protected readonly note = new FormControl<string>('', { nonNullable: true });

  protected readonly form = new FormGroup({
    direction: this.direction,
    currency: this.currency,
    amount: this.amount,
    counterpartyName: this.counterpartyName,
    note: this.note,
  });

  protected readonly currencies: CurrencyCode[] = ['EUR', 'USD'];

  constructor() {
    super();
    this.$defaultBackUrlSpacePath.set('debts');
    this.route.queryParamMap.pipe(first(), takeUntilDestroyed()).subscribe({
      next: (params) => {
        const dir = params.get('direction');
        if (dir === 'lend' || dir === 'borrow') {
          this.direction.setValue(dir);
        }
        const contactID = params.get('contactID');
        if (contactID) {
          this.prefilledContactID = contactID;
          // Prefill the name field from the known balance so the user sees who
          // they're recording against even before the contactus picker loads.
          this.spaceIDChanged$.pipe(first()).subscribe((spaceID) => {
            this.debtusService
              .getContactBalance(spaceID ?? '', contactID)
              .pipe(first())
              .subscribe((c) => this.counterpartyName.setValue(c.title));
          });
        }
      },
    });
  }

  protected onContactChanged(contact: IContactContext | undefined): void {
    this.$pickedContact.set(contact);
    if (contact) {
      // A picked contact supersedes a typed name / prefilled id.
      this.prefilledContactID = undefined;
      this.counterpartyName.setValue(contact.brief?.title ?? '');
    }
  }

  private resolveCounterparty(): { contactID: string; title: string } | null {
    const picked = this.$pickedContact();
    if (picked) {
      return { contactID: picked.id, title: picked.brief?.title ?? picked.id };
    }
    if (this.prefilledContactID) {
      return {
        contactID: this.prefilledContactID,
        title: this.counterpartyName.value,
      };
    }
    const name = this.counterpartyName.value.trim();
    if (name) {
      // New counterparty by name (mirrors the bot's "new counterparty" step).
      // No contactID yet — the backend create flow resolves/creates it.
      return { contactID: '', title: name };
    }
    return null;
  }

  protected submit(): void {
    this.form.markAllAsTouched();
    const amount = this.amount.value;
    if (!amount || amount <= 0) {
      return;
    }
    const counterparty = this.resolveCounterparty();
    if (!counterparty) {
      this.showToast('Pick a contact or enter a counterparty name.', 'danger');
      return;
    }
    const spaceID = this.$spaceID();
    if (!spaceID) {
      return;
    }
    const request: ICreateTransferRequest = {
      spaceID,
      direction: this.direction.value,
      amount: { currency: this.currency.value, value: amount },
      contactID: counterparty.contactID,
      contactTitle: counterparty.title,
      note: this.note.value.trim() || undefined,
      counterpartySpaceID: this.$pickedContact()?.space?.id,
    };

    this.$submitting.set(true);
    this.debtusService.createTransfer(request).subscribe({
      next: (resp) => {
        this.$submitting.set(false);
        // Navigation default: go to the created transfer's detail, replaceUrl
        // so Back doesn't reopen the filled form.
        this.spaceNav.navigateForwardToSpacePage(
          this.space,
          `transfer/${resp.transfer.id}`,
          { replaceUrl: true },
        );
      },
      error: (err) => {
        this.$submitting.set(false);
        this.errorLogger.logError(err, 'Failed to create transfer', {
          show: false,
        });
        this.showToast('Failed to record transfer. Please try again.', 'danger');
      },
    });
  }

  private async showToast(message: string, color: string): Promise<void> {
    const toast = await this.toastController.create({
      message,
      duration: 3000,
      color,
    });
    await toast.present();
  }
}
