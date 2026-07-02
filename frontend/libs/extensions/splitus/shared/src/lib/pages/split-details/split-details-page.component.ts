import { Component, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import {
  IonBackButton,
  IonBadge,
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
import { ISplit, SPLITUS_SERVICE } from '@sneat/extension-splitus-contract';
import {
  SpaceComponentBaseParams,
  SpacePageBaseComponent,
} from '@sneat/space-components';
import { ClassName } from '@sneat/ui';
import { combineLatest, switchMap } from 'rxjs';

// Minimal split details page — the landing target of the create-split
// redirect (flows.md: a create flow must exit onto a real screen). Shows the
// split title/total and its raw participant list from getSplit; the full
// details experience is a later task. Loads from the :splitID in the URL, so
// deep links and refreshes work without router state.
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
}
