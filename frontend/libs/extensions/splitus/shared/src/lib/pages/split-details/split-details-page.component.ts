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
  IonItem,
  IonLabel,
  IonList,
  IonNote,
  IonSpinner,
  IonTitle,
  IonToolbar,
} from '@ionic/angular/standalone';
import { ISplit, ISplitParticipant, SPLITUS_SERVICE } from '@sneat/extension-splitus-contract';
import {
  SpaceComponentBaseParams,
  SpacePageBaseComponent,
} from '@sneat/space-components';
import { ClassName } from '@sneat/ui';
import { combineLatest, switchMap } from 'rxjs';
import { settleInDebtusUrl, statusColor } from '../split-status-view';

// Split details — per-participant shares with settled/outstanding read
// straight from getSplit (splitus#ac:settle-up-single-source-of-truth: no
// client-side settled logic or storage), plus a "Settle in Debtus" link on
// outstanding rows. Loads by the :splitID in the URL, so deep links and
// refreshes work without router state. Entry: the splits list
// (splitus-home-page) and new-split's post-create redirect. Exit: back to
// `splits`, or out to debtus.app to actually settle.
@Component({
  selector: 'sneat-splitus-split-details-page',
  templateUrl: './split-details-page.component.html',
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
    IonSpinner,
  ],
  providers: [
    { provide: ClassName, useValue: 'SplitDetailsPageComponent' },
    SpaceComponentBaseParams,
  ],
})
export class SplitDetailsPageComponent extends SpacePageBaseComponent {
  private readonly splitusService = inject(SPLITUS_SERVICE);

  protected readonly $loading = signal(true);
  protected readonly $error = signal<string | undefined>(undefined);
  protected readonly $split = signal<ISplit | undefined>(undefined);

  protected readonly statusColor = statusColor;

  constructor() {
    super();
    this.$defaultBackUrlSpacePath.set('splits');
    combineLatest([this.spaceIDChanged$, this.route.paramMap])
      .pipe(
        switchMap(([spaceID, params]) => {
          const splitID = params.get('splitID') ?? '';
          this.$loading.set(true);
          this.$error.set(undefined);
          return this.splitusService.getSplit(spaceID ?? '', splitID);
        }),
        takeUntilDestroyed(),
      )
      .subscribe({
        next: (split) => {
          this.$split.set(split);
          this.$loading.set(false);
        },
        error: (err) => {
          this.$error.set('Failed to load split');
          this.$loading.set(false);
          this.errorLogger.logError(err, 'Failed to load split', {
            show: false,
          });
        },
      });
  }

  protected settleUrl(participant: ISplitParticipant): string | undefined {
    return settleInDebtusUrl(
      participant,
      this.$spaceType(),
      this.$spaceID(),
    );
  }
}
