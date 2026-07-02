import { Component, Input, inject } from '@angular/core';
import {
  FormControl,
  FormGroup,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import {
  IonButton,
  IonCard,
  IonCardHeader,
  IonCardTitle,
  IonInput,
  IonItem,
  IonLabel,
  IonSelect,
  IonSelectOption,
} from '@ionic/angular/standalone';
import { IContactContext } from '@sneat/extension-contactus-contract';
import { ErrorLogger, IErrorLogger } from '@sneat/core';
import {
  CurrencyCode,
  ICreateSplitRequest,
  SPLITUS_SERVICE,
} from '@sneat/extension-splitus-contract';
import { ISpaceContext } from '@sneat/space-models';

@Component({
  selector: 'sneat-splitus-new-split-form',
  templateUrl: './new-split-form.component.html',
  imports: [
    ReactiveFormsModule,
    IonCard,
    IonCardHeader,
    IonCardTitle,
    IonItem,
    IonLabel,
    IonSelect,
    IonSelectOption,
    IonInput,
    IonButton,
    // ContactInputComponent,
    // forwardRef(() => ContactInputComponent),
  ],
})
export class NewSplitFormComponent {
  private readonly errorLogger = inject<IErrorLogger>(ErrorLogger);
  private readonly splitusService = inject(SPLITUS_SERVICE);

  @Input({ required: true }) public space?: ISpaceContext;
  @Input({ required: true }) public contact?: IContactContext;

  protected currency = new FormControl<CurrencyCode>('EUR');
  protected amount = new FormControl<number | undefined>(
    undefined,
    Validators.required,
  );

  protected newSplitForm = new FormGroup({
    currency: this.currency,
    amount: this.amount,
  });

  protected readonly currencies = ['EUR', 'USD'];

  protected submit() {
    const spaceID = this.space?.id;
    if (!spaceID) {
      throw new Error('spaceID is not set');
    }
    const contactID = this.contact?.id;
    if (!contactID) {
      throw new Error('contactID is not set');
    }
    if (!this.amount.value) {
      throw new Error('amount is not set');
    }
    if (!this.currency.value) {
      throw new Error('currency is not set');
    }
    // TODO(later task): this form only collects a single contact today; the
    // full equal/exact/percentage split screen with multiple participants is
    // a later task. This wiring is kept minimal — one participant, equal
    // split — just enough to compile against the real createSplit contract.
    const request: ICreateSplitRequest = {
      spaceID,
      currency: this.currency.value,
      amount: this.amount.value.toFixed(2),
      participantContactIDs: [contactID],
    };
    this.splitusService.createSplit(request).subscribe({
      next: () => {
        // Split created successfully
      },
      error: (err) => {
        console.error('Failed to create split:', err);
      },
    });
  }
}
