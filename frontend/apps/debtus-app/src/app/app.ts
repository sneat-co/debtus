import { Component } from '@angular/core';
import { IonApp, IonRouterOutlet } from '@ionic/angular/standalone';
import { BaseAppComponent } from '@sneat/app';

@Component({
  selector: 'debtus-root',
  template: '<ion-app><ion-router-outlet /></ion-app>',
  imports: [IonApp, IonRouterOutlet],
})
export class App extends BaseAppComponent {}
