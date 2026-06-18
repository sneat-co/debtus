import { Component } from '@angular/core';
import { RouterLink } from '@angular/router';
import {
  IonApp,
  IonContent,
  IonHeader,
  IonMenu,
  IonRouterOutlet,
  IonSplitPane,
  IonTitle,
  IonToolbar,
} from '@ionic/angular/standalone';
import { BaseAppComponent } from '@sneat/app';
import { AuthMenuItemComponent } from '@sneat/auth-ui';

// Extends BaseAppComponent for the shared app lifecycle (redirect sign-in
// completion, title strategy, analytics, current-space clearing). Hosts a side
// menu (like sneat-app) whose sneat-auth-menu-item shows the signed-in user and
// a sign-out action.
@Component({
  selector: 'debtus-root',
  template: `
    <ion-app>
      <ion-split-pane contentId="main">
        <ion-menu contentId="main" #menu>
          <ion-header>
            <ion-toolbar color="light">
              <ion-title [routerLink]="'/'" tappable (click)="menu.close()">
                Debtus.app
              </ion-title>
            </ion-toolbar>
          </ion-header>
          <ion-content>
            <sneat-auth-menu-item />
          </ion-content>
        </ion-menu>
        <ion-router-outlet id="main" />
      </ion-split-pane>
    </ion-app>
  `,
  imports: [
    IonApp,
    IonSplitPane,
    IonMenu,
    IonHeader,
    IonToolbar,
    IonTitle,
    IonContent,
    IonRouterOutlet,
    RouterLink,
    AuthMenuItemComponent,
  ],
})
export class App extends BaseAppComponent {}
