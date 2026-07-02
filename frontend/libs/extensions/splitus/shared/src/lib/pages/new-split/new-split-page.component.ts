import { Component } from '@angular/core';
import {
  IonBackButton,
  IonButtons,
  IonContent,
  IonHeader,
  IonTitle,
  IonToolbar,
} from '@ionic/angular/standalone';
import {
  SpaceComponentBaseParams,
  SpacePageBaseComponent,
} from '@sneat/space-components';
import { ClassName } from '@sneat/ui';
import { NewSplitFormComponent } from '../../components';

// Create-split page. Entry: the "New split" button on the splits home page.
// Exit: the hosted form redirects to `split/:splitID` (details) on success
// with replaceUrl; Back/cancel returns to `splits`.
@Component({
  selector: 'sneat-splitus-new-split-page',
  templateUrl: './new-split-page.component.html',
  imports: [
    IonHeader,
    IonToolbar,
    IonButtons,
    IonBackButton,
    IonTitle,
    IonContent,
    NewSplitFormComponent,
  ],
  providers: [
    { provide: ClassName, useValue: 'NewSplitPageComponent' },
    SpaceComponentBaseParams,
  ],
})
export class NewSplitPageComponent extends SpacePageBaseComponent {
  constructor() {
    super();
    this.$defaultBackUrlSpacePath.set('splits');
  }
}
