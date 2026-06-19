import { Component } from '@angular/core';
import {
  IonBackButton,
  IonButtons,
  IonContent,
  IonHeader,
  IonTitle,
  IonToolbar,
} from '@ionic/angular/standalone';
import { UserAuthAccountsComponent } from '@sneat/auth-ui';
import { UserCountryComponent } from '@sneat/components';

// /my profile page for splitus. Reuses the shared, published profile pieces:
// the user's linked auth accounts (sneat-user-auth-accounts) and country
// (sneat-user-country). A lighter equivalent of sneat-app's UserMyProfilePage.
@Component({
  selector: 'splitus-my-profile-page',
  imports: [
    IonHeader,
    IonToolbar,
    IonButtons,
    IonBackButton,
    IonTitle,
    IonContent,
    UserAuthAccountsComponent,
    UserCountryComponent,
  ],
  template: `
    <ion-header>
      <ion-toolbar color="light">
        <ion-buttons slot="start">
          <ion-back-button defaultHref="/" />
        </ion-buttons>
        <ion-title>My profile @ Splitus.app</ion-title>
      </ion-toolbar>
    </ion-header>
    <ion-content class="ion-padding">
      <sneat-user-country [doNotHide]="true" />
      <sneat-user-auth-accounts />
    </ion-content>
  `,
})
export class MyProfilePageComponent {}
