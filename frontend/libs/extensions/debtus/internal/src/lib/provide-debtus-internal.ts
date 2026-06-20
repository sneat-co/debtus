import { Provider } from '@angular/core';
import { DEBTUS_SERVICE } from '@sneat/extension-debtus-contract';
import { DebtusService } from './services';

// Registers the concrete DebtusService and binds it to the DEBTUS_SERVICE token so
// consumers depend only on the IDebtusService contract. Wired in at app
// bootstrap (consumers do not import this factory directly).
export function provideDebtusInternal(): Provider[] {
  return [DebtusService, { provide: DEBTUS_SERVICE, useExisting: DebtusService }];
}
