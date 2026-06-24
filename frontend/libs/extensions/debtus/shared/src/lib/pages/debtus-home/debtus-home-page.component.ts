import { Component } from '@angular/core';
import {
  IonBackButton,
  IonButtons,
  IonContent,
  IonHeader,
  IonToolbar,
} from '@ionic/angular/standalone';
import { ContactusServicesModule } from '@sneat/extension-contactus-internal';
import { NewDebtFormComponent } from '../../components';
import {
  SpaceComponentBaseParams,
  SpacePageTitleComponent,
  SpacePageBaseComponent,
} from '@sneat/space-components';
import { SpaceServiceModule } from '@sneat/space-services';
import { ClassName } from '@sneat/ui';

@Component({
  selector: 'sneat-debtus-home-page',
  templateUrl: './debtus-home-page.component.html',
  imports: [
    SpacePageTitleComponent,
    NewDebtFormComponent,
    ContactusServicesModule,
    SpaceServiceModule,
    IonHeader,
    IonToolbar,
    IonButtons,
    IonBackButton,
    IonContent,
  ],
  providers: [
    {
      provide: ClassName,
      useValue: 'DebtusHomePageComponent',
    },
    SpaceComponentBaseParams,
  ],
})
export class DebtusHomePageComponent extends SpacePageBaseComponent {}
