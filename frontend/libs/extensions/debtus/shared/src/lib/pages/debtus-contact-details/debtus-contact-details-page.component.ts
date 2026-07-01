import { DatePipe } from '@angular/common';
import { Component, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import {
  IonBackButton,
  IonBadge,
  IonButton,
  IonButtons,
  IonCard,
  IonCardContent,
  IonCardHeader,
  IonCardTitle,
  IonContent,
  IonHeader,
  IonIcon,
  IonItem,
  IonLabel,
  IonList,
  IonNote,
  IonSpinner,
  IonTitle,
  IonToolbar,
  ToastController,
} from '@ionic/angular/standalone';
import {
  DEBTUS_SERVICE,
  IContactBalance,
  IDebtusTransfer,
  formatAmount,
  formatSignedBalance,
} from '@sneat/extension-debtus-contract';
import {
  SpaceComponentBaseParams,
  SpacePageBaseComponent,
} from '@sneat/space-components';
import { combineLatest, switchMap } from 'rxjs';

// Contact detail — balance with one counterparty plus their transfer history.
// Actions mirror the bot: record a lend/borrow, remind, and settle up.
@Component({
  selector: 'sneat-debtus-contact-details-page',
  templateUrl: './debtus-contact-details-page.component.html',
  imports: [
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
    IonList,
    IonItem,
    IonLabel,
    IonNote,
    IonBadge,
    IonButton,
    IonIcon,
    IonSpinner,
    DatePipe,
  ],
  providers: [SpaceComponentBaseParams],
})
export class DebtusContactDetailsPageComponent extends SpacePageBaseComponent {
  private readonly debtusService = inject(DEBTUS_SERVICE);
  private readonly toastController = inject(ToastController);

  protected readonly $loading = signal(true);
  protected readonly $error = signal<string | undefined>(undefined);
  protected readonly $contact = signal<IContactBalance | undefined>(undefined);
  protected readonly $transfers = signal<IDebtusTransfer[]>([]);
  protected contactID = '';

  protected readonly formatAmount = formatAmount;
  protected readonly formatSignedBalance = formatSignedBalance;

  constructor() {
    super();
    combineLatest([this.spaceIDChanged$, this.route.paramMap])
      .pipe(
        switchMap(([spaceID, params]) => {
          this.contactID = params.get('contactID') ?? '';
          this.$loading.set(true);
          this.$error.set(undefined);
          return combineLatest([
            this.debtusService.getContactBalance(
              spaceID ?? '',
              this.contactID,
            ),
            this.debtusService.getTransfers(spaceID ?? '', this.contactID),
          ]);
        }),
        takeUntilDestroyed(),
      )
      .subscribe({
        next: ([contact, transfers]) => {
          this.$contact.set(contact);
          this.$transfers.set(transfers);
          this.$loading.set(false);
        },
        error: (err) => {
          this.$error.set('Failed to load contact');
          this.$loading.set(false);
          this.errorLogger.logError(err, 'Failed to load debtus contact');
        },
      });
  }

  protected recordLend(): void {
    this.spaceNav.navigateForwardToSpacePage(
      this.space,
      `new-transfer?direction=lend&contactID=${this.contactID}`,
    );
  }

  protected recordBorrow(): void {
    this.spaceNav.navigateForwardToSpacePage(
      this.space,
      `new-transfer?direction=borrow&contactID=${this.contactID}`,
    );
  }

  protected settleUp(): void {
    this.spaceNav.navigateForwardToSpacePage(
      this.space,
      `settle-up?contactID=${this.contactID}`,
    );
  }

  protected async remind(): Promise<void> {
    // Fable: prototype demo data — no wired reminder-send HTTP endpoint yet.
    // The bot schedules reminders via a background cron; here we just confirm
    // the intent. SWAP: call a POST reminder endpoint once available.
    const toast = await this.toastController.create({
      message: `Reminder queued for ${this.$contact()?.title ?? 'contact'} (demo).`,
      duration: 2500,
      color: 'medium',
    });
    await toast.present();
  }

  protected openTransfer(transferID: string): void {
    this.spaceNav.navigateForwardToSpacePage(
      this.space,
      `transfer/${transferID}`,
    );
  }
}
