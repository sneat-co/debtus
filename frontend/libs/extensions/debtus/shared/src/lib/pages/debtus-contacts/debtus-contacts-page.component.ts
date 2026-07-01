import { Component, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import {
  IonBackButton,
  IonButtons,
  IonContent,
  IonHeader,
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
  IContactBalance,
  formatSignedBalance,
} from '@sneat/extension-debtus-contract';
import {
  SpaceComponentBaseParams,
  SpacePageBaseComponent,
} from '@sneat/space-components';
import { ClassName } from '@sneat/ui';
import { switchMap } from 'rxjs';

// Contacts list — counterparties (contactus space contacts) with their net
// balance. Tap a row to open the per-contact detail.
@Component({
  selector: 'sneat-debtus-contacts-page',
  templateUrl: './debtus-contacts-page.component.html',
  imports: [
    IonHeader,
    IonToolbar,
    IonButtons,
    IonBackButton,
    IonTitle,
    IonContent,
    IonList,
    IonItem,
    IonLabel,
    IonNote,
    IonSpinner,
  ],
  providers: [
    { provide: ClassName, useValue: 'DebtusContactsPageComponent' },
    SpaceComponentBaseParams,
  ],
})
export class DebtusContactsPageComponent extends SpacePageBaseComponent {
  private readonly debtusService = inject(DEBTUS_SERVICE);

  protected readonly $loading = signal(true);
  protected readonly $error = signal<string | undefined>(undefined);
  protected readonly $contacts = signal<IContactBalance[]>([]);

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
          this.$contacts.set(contacts);
          this.$loading.set(false);
        },
        error: (err) => {
          this.$error.set('Failed to load contacts');
          this.$loading.set(false);
          this.errorLogger.logError(err, 'Failed to load debtus contacts');
        },
      });
  }

  protected openContact(contactID: string): void {
    this.spaceNav.navigateForwardToSpacePage(
      this.space,
      `debtus-contact/${contactID}`,
    );
  }
}
