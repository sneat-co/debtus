import { Provider } from '@angular/core';
import { SPLITUS_SERVICE } from '@sneat/extension-splitus-contract';
import { SplitusService } from './services';

// Registers the concrete SplitusService and binds it to the SPLITUS_SERVICE token so
// consumers depend only on the ISplitusService contract. Wired in at app
// bootstrap (consumers do not import this factory directly).
export function provideSplitusInternal(): Provider[] {
  return [SplitusService, { provide: SPLITUS_SERVICE, useExisting: SplitusService }];
}
