import { Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import {
  IonContent,
  IonHeader,
  IonItem,
  IonLabel,
  IonList,
  IonListHeader,
  IonNote,
  IonSpinner,
  IonTitle,
  IonToolbar,
} from '@ionic/angular/standalone';
import { SneatUserService } from '@sneat/auth-core';

// Authenticated landing page for debtus.app. Lists the spaces the signed-in
// user belongs to (read from their user record) and links into each space.
// Without this, the root route redirected to /login, so after sign-in the app
// bounced authenticated users straight back to the login page.
@Component({
  selector: 'debtus-home-page',
  imports: [
    IonHeader,
    IonToolbar,
    IonTitle,
    IonContent,
    IonList,
    IonListHeader,
    IonItem,
    IonLabel,
    IonNote,
    IonSpinner,
    RouterLink,
  ],
  template: `
    <ion-header>
      <ion-toolbar>
        <ion-title>debtus.app</ion-title>
      </ion-toolbar>
    </ion-header>
    <ion-content class="ion-padding">
      @if (spaces() === undefined) {
        <ion-spinner aria-label="Loading your spaces" />
      } @else if (spaces()!.length === 0) {
        <p>You don't belong to any spaces yet.</p>
      } @else {
        <ion-list>
          <ion-list-header>Your spaces</ion-list-header>
          @for (space of spaces(); track space.id) {
            <ion-item button [routerLink]="['/space', space.type, space.id]">
              <ion-label>{{ space.title || space.id }}</ion-label>
              <ion-note slot="end">{{ space.type }}</ion-note>
            </ion-item>
          }
        </ion-list>
      }
    </ion-content>
  `,
})
export class DebtusHomePageComponent {
  private readonly userService = inject(SneatUserService);
  private readonly userState = toSignal(this.userService.userState);

  // undefined => still loading the user record; [] => loaded, no spaces.
  protected readonly spaces = computed(() => {
    const record = this.userState()?.record;
    if (!record) {
      return undefined;
    }
    const spaces = record.spaces ?? {};
    return Object.entries(spaces).map(([id, brief]) => ({ id, ...brief }));
  });
}
