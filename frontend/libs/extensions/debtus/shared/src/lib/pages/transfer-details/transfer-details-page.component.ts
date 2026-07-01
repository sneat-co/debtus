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
} from '@ionic/angular/standalone';
import {
  DEBTUS_SERVICE,
  IDebtusTransfer,
  formatAmount,
} from '@sneat/extension-debtus-contract';
import {
  SpaceComponentBaseParams,
  SpacePageBaseComponent,
} from '@sneat/space-components';
import { combineLatest, switchMap } from 'rxjs';

// Transfer detail / receipt — amount, parties, date, note, and its effect on
// the balance. Mirrors the bot's receipt. Offers a settle-up entry.
@Component({
  selector: 'sneat-debtus-transfer-details-page',
  templateUrl: './transfer-details-page.component.html',
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
export class TransferDetailsPageComponent extends SpacePageBaseComponent {
  private readonly debtusService = inject(DEBTUS_SERVICE);

  protected readonly $loading = signal(true);
  protected readonly $error = signal<string | undefined>(undefined);
  protected readonly $transfer = signal<IDebtusTransfer | undefined>(undefined);

  protected readonly formatAmount = formatAmount;

  constructor() {
    super();
    this.$defaultBackUrlSpacePath.set('debts');
    combineLatest([this.spaceIDChanged$, this.route.paramMap])
      .pipe(
        switchMap(([spaceID, params]) => {
          const transferID = params.get('transferID') ?? '';
          this.$loading.set(true);
          this.$error.set(undefined);
          return this.debtusService.getTransfer(spaceID ?? '', transferID);
        }),
        takeUntilDestroyed(),
      )
      .subscribe({
        next: (transfer) => {
          this.$transfer.set(transfer);
          this.$loading.set(false);
        },
        error: (err) => {
          this.$error.set('Failed to load transfer');
          this.$loading.set(false);
          this.errorLogger.logError(err, 'Failed to load debtus transfer');
        },
      });
  }

  protected settleUp(): void {
    const t = this.$transfer();
    if (!t) {
      return;
    }
    this.spaceNav.navigateForwardToSpacePage(
      this.space,
      `settle-up?contactID=${t.counterpartyContactID}`,
    );
  }

  protected openContact(): void {
    const t = this.$transfer();
    if (!t?.counterpartyContactID) {
      return;
    }
    this.spaceNav.navigateForwardToSpacePage(
      this.space,
      `debtus-contact/${t.counterpartyContactID}`,
    );
  }
}
