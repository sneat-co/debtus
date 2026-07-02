import { Component, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import {
  IonBackButton,
  IonBadge,
  IonButton,
  IonButtons,
  IonCard,
  IonCardContent,
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
import { ISplitListItem, SPLITUS_SERVICE } from '@sneat/extension-splitus-contract';
import {
  SpaceComponentBaseParams,
  SpacePageTitleComponent,
  SpacePageBaseComponent,
} from '@sneat/space-components';
import { SpaceServiceModule } from '@sneat/space-services';
import { ClassName } from '@sneat/ui';
import { switchMap } from 'rxjs';
import { formatMemberCount, statusColor } from '../split-status-view';

// Splits home — the splitus entry point for a space: the space's splits
// (read via getSplits) plus the "New split" action. Entry: space navigation /
// the splitus-space default redirect. Exit: "New split" -> new-split; each
// row -> split/:splitID.
@Component({
  selector: 'sneat-splitus-home-page',
  templateUrl: './splitus-home-page.component.html',
  imports: [
    SpacePageTitleComponent,
    SpaceServiceModule,
    IonHeader,
    IonToolbar,
    IonButtons,
    IonBackButton,
    IonContent,
    IonCard,
    IonCardContent,
    IonButton,
    IonIcon,
    IonList,
    IonItem,
    IonLabel,
    IonNote,
    IonBadge,
    IonSpinner,
  ],
  providers: [
    {
      provide: ClassName,
      useValue: 'SplitusHomePageComponent',
    },
    SpaceComponentBaseParams,
  ],
})
export class SplitusHomePageComponent extends SpacePageBaseComponent {
  private readonly splitusService = inject(SPLITUS_SERVICE);

  protected readonly $loading = signal(true);
  protected readonly $error = signal<string | undefined>(undefined);
  protected readonly $splits = signal<ISplitListItem[]>([]);

  protected readonly statusColor = statusColor;
  protected readonly formatMemberCount = formatMemberCount;

  constructor() {
    super();
    this.spaceIDChanged$
      .pipe(
        switchMap((spaceID) => {
          this.$loading.set(true);
          this.$error.set(undefined);
          return this.splitusService.getSplits(spaceID ?? '');
        }),
        takeUntilDestroyed(),
      )
      .subscribe({
        next: (splits) => {
          this.$splits.set(splits);
          this.$loading.set(false);
        },
        error: (err) => {
          this.$error.set('Failed to load splits');
          this.$loading.set(false);
          this.errorLogger.logError(err, 'Failed to load splits');
        },
      });
  }

  protected goNewSplit(): void {
    this.spaceNav.navigateForwardToSpacePage(this.space, 'new-split');
  }

  protected openSplit(splitID: string): void {
    this.spaceNav.navigateForwardToSpacePage(this.space, `split/${splitID}`);
  }
}
