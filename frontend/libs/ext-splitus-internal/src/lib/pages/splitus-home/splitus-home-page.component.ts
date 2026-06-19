import { Component } from '@angular/core';
import {
  IonBackButton,
  IonButtons,
  IonContent,
  IonHeader,
  IonToolbar,
} from '@ionic/angular/standalone';
import { ContactusServicesModule } from '@sneat/contactus-services';
import { NewSplitFormComponent } from '@sneat/ext-splitus-shared';
import {
  SpaceComponentBaseParams,
  SpacePageTitleComponent,
  SpacePageBaseComponent,
} from '@sneat/space-components';
import { SpaceServiceModule } from '@sneat/space-services';
import { ClassName } from '@sneat/ui';

@Component({
  selector: 'sneat-splitus-home-page',
  templateUrl: './splitus-home-page.component.html',
  imports: [
    SpacePageTitleComponent,
    NewSplitFormComponent,
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
      useValue: 'SplitusHomePageComponent',
    },
    SpaceComponentBaseParams,
  ],
})
export class SplitusHomePageComponent extends SpacePageBaseComponent {}
