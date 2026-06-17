import { Component } from '@angular/core';
import {
  IonContent,
  IonHeader,
  IonTitle,
  IonToolbar,
} from '@ionic/angular/standalone';
import { SpacesCardComponent } from '@sneat/space-components';

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
