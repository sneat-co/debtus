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
  IonToolbar,
} from '@ionic/angular/standalone';
import { DemoDataBannerComponent } from '../../components/demo-data-banner/demo-data-banner.component';
import {
  DEBTUS_SERVICE,
  IBalanceSummary,
  IContactBalance,
  formatAmount,
  formatSignedBalance,
  isZeroBalance,
  summarizeBalances,
} from '@sneat/extension-debtus-contract';
import {
  SpaceComponentBaseParams,
  SpacePageBaseComponent,
  SpacePageTitleComponent,
} from '@sneat/space-components';
import { SpaceServiceModule } from '@sneat/space-services';
import { ClassName } from '@sneat/ui';
import { switchMap } from 'rxjs';

// Home / balances screen — the debtus entry point. Mirrors the bot's
// `debts_balance` view: a net summary of who owes the user and whom the user
// owes, plus quick actions into contacts and creating a transfer.
@Component({
  selector: 'sneat-debtus-home-page',
  templateUrl: './debtus-home-page.component.html',
  imports: [
    DemoDataBannerComponent,
    SpacePageTitleComponent,
    SpaceServiceModule,
    IonHeader,
    IonToolbar,
    IonButtons,
    IonBackButton,
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
  ],
  providers: [
    { provide: ClassName, useValue: 'DebtusHomePageComponent' },
    SpaceComponentBaseParams,
  ],
})
export class DebtusHomePageComponent extends SpacePageBaseComponent {
  private readonly debtusService = inject(DEBTUS_SERVICE);

  protected readonly $loading = signal(true);
  protected readonly $error = signal<string | undefined>(undefined);
  protected readonly $contacts = signal<IContactBalance[]>([]);
  protected readonly $summary = signal<IBalanceSummary>({
    theyOweYou: [],
    youOwe: [],
  });

  protected readonly formatAmount = formatAmount;
  protected readonly formatSignedBalance = formatSignedBalance;

  constructor() {
    super();
    this.spaceIDChanged$
      .pipe(
        switchMap((spaceID) => {
          this.$loading.set(true);
          this.$error.set(undefined);
          return this.debtusService.getContactBalances(spaceID ?? '');
        }),
        takeUntilDestroyed(),
      )
      .subscribe({
        next: (contacts) => {
          const active = contacts.filter((c) => !isZeroBalance(c.balance));
          this.$contacts.set(active);
          this.$summary.set(summarizeBalances(contacts));
          this.$loading.set(false);
        },
        error: (err) => {
          this.$error.set('Failed to load balances');
          this.$loading.set(false);
          this.errorLogger.logError(err, 'Failed to load contact balances');
        },
      });
  }

  protected goContacts(): void {
    this.spaceNav.navigateForwardToSpacePage(this.space, 'debtus-contacts');
  }

  protected goNewTransfer(): void {
    this.spaceNav.navigateForwardToSpacePage(this.space, 'new-transfer');
  }

  protected goContact(contactID: string): void {
    this.spaceNav.navigateForwardToSpacePage(
      this.space,
      `debtus-contact/${contactID}`,
    );
  }
}
