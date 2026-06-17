// Main entry point for debtus.app
import { bootstrapApplication } from '@angular/platform-browser';
import { provideRouter } from '@angular/router';
import {
  getStandardSneatProviders,
  provideAppInfo,
  provideRolesByType,
} from '@sneat/app';
import { SneatAuthStateService } from '@sneat/auth-core';
import { authRoutes } from '@sneat/auth-ui';
import { App } from './app/app';
import { appRoutes } from './app/app.routes';
import { DebtusAuthStateService } from './app/debtus-auth-state.service';
import { debtusAppEnvironmentConfig } from './environments/environment';
import { registerIonicons } from './register-ionicons';

bootstrapApplication(App, {
  providers: [
    ...getStandardSneatProviders(debtusAppEnvironmentConfig),
    provideAppInfo({ appId: 'debtus', appTitle: 'debtus.app' }),
    provideRouter([...appRoutes, ...authRoutes]),
    provideRolesByType(undefined),
    // Use redirect-based sign-in on debtus.app (popup hangs under Google's COOP).
    { provide: SneatAuthStateService, useClass: DebtusAuthStateService },
  ],
}).catch((err) => console.error(err));

registerIonicons();
