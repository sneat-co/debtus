import { Component } from '@angular/core';
import {
  IonContent,
  IonHeader,
  IonTitle,
  IonToolbar,
} from '@ionic/angular/standalone';
import { SpacesCardComponent } from '@sneat/space-components';
import { SpaceService } from '@sneat/space-services';

// Authenticated landing page for debtus.app. Reuses the shared
// SpacesCardComponent (the same component sneat-app uses to list a user's
// spaces): it watches the signed-in user's record, renders their spaces with
// proper titles, and links into each space. Without an authed landing the root
// route redirected to /login and bounced signed-in users back to the login page.
@Component({
  selector: 'debtus-home-page',
  imports: [
    IonHeader,
    IonToolbar,
    IonTitle,
    IonContent,
    SpacesCardComponent,
  ],
  // SpaceService is @Injectable() (provided via SpaceServiceModule), not
  // providedIn:'root'. debtus only provides it inside its space routes, so the
  // root-level home page must provide it for SpacesCardComponent to resolve it.
  providers: [SpaceService],
  template: `
    <ion-header>
      <ion-toolbar>
        <ion-title>debtus.app</ion-title>
      </ion-toolbar>
    </ion-header>
    <ion-content class="ion-padding">
      <sneat-spaces-card />
    </ion-content>
  `,
})
export class DebtusHomePageComponent {}
