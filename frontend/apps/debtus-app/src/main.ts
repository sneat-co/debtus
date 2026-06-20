// Main entry point for debtus.app
import { bootstrapApplication } from '@angular/platform-browser';
import { provideRouter } from '@angular/router';
import {
  getStandardSneatProviders,
  provideAppInfo,
  provideRolesByType,
} from '@sneat/app';
import { authRoutes } from '@sneat/auth-ui';
import { provideDebtusInternal } from '@sneat/extension-debtus-internal';
import { App } from './app/app';
import { appRoutes } from './app/app.routes';
import { debtusAppEnvironmentConfig } from './environments/environment';
import { registerIonicons } from './register-ionicons';

bootstrapApplication(App, {
  providers: [
    ...getStandardSneatProviders(debtusAppEnvironmentConfig),
    // Bind the debtus contract tokens (DEBTUS_SERVICE) to their concrete
    // implementations. The app is the composition root and may wire -internal.
    ...provideDebtusInternal(),
    provideAppInfo({ appId: 'debtus', appTitle: 'Debtus.app' }),
    provideRouter([...appRoutes, ...authRoutes]),
    provideRolesByType(undefined),
  ],
}).catch((err) => console.error(err));

registerIonicons();
