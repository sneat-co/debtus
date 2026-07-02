import { Component } from '@angular/core';
import {
  IonBackButton,
  IonButton,
  IonButtons,
  IonCard,
  IonCardContent,
  IonContent,
  IonHeader,
  IonIcon,
  IonToolbar,
} from '@ionic/angular/standalone';
import {
  SpaceComponentBaseParams,
  SpacePageTitleComponent,
  SpacePageBaseComponent,
} from '@sneat/space-components';
import { SpaceServiceModule } from '@sneat/space-services';
import { ClassName } from '@sneat/ui';

// Splits home — the splitus entry point for a space. Offers the "New split"
// action (→ new-split); the splits list itself is a later task.
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
  protected goNewSplit(): void {
    this.spaceNav.navigateForwardToSpacePage(this.space, 'new-split');
  }
}
