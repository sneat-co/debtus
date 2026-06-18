import { Component } from '@angular/core';
import {
  IonButtons,
  IonContent,
  IonHeader,
  IonMenuButton,
  IonTitle,
  IonToolbar,
} from '@ionic/angular/standalone';
import { UserRequiredFieldsService } from '@sneat/auth-ui';
import { SpacesCardComponent } from '@sneat/space-components';
import { SpaceService } from '@sneat/space-services';

// Authenticated landing page for debtus.app. Reuses the shared
// SpacesCardComponent to list the user's spaces. The menu button opens the side
// menu (in the app shell) which shows the signed-in user + sign-out.
@Component({
  selector: 'debtus-home-page',
  imports: [
    IonHeader,
    IonToolbar,
    IonTitle,
    IonContent,
    IonButtons,
    IonMenuButton,
    SpacesCardComponent,
  ],
  // SpaceService and UserRequiredFieldsService are @Injectable() (not
  // providedIn:'root' before @sneat 0.9.1). The embedded SpacesCard -> SpacesList
  // chain needs both, so this root-level landing page provides them.
  providers: [SpaceService, UserRequiredFieldsService],
  template: `
    <ion-header>
      <ion-toolbar>
        <ion-buttons slot="start">
          <ion-menu-button />
        </ion-buttons>
        <ion-title>Debtus.app</ion-title>
      </ion-toolbar>
    </ion-header>
    <ion-content class="ion-padding">
      <sneat-spaces-card />
    </ion-content>
  `,
})
export class DebtusHomePageComponent {}
