// Main entry point for debtus.app
import { bootstrapApplication } from '@angular/platform-browser';
import { provideRouter } from '@angular/router';
import {
  getStandardSneatProviders,
  provideAppInfo,
  provideRolesByType,
} from '@sneat/app';
import { authRoutes } from '@sneat/auth-ui';
import { provideContactusInternal } from '@sneat/extension-contactus-internal';
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
    // The debtus counterparty picker reuses the contactus space-contact
    // selector (sneat-contact-input), which needs the contactus services wired
    // at the composition root. Requires @sneat/extension-contactus-internal
    // >= 0.12.3 (0.12.2 crashes at bootstrap with NG0204).
    ...provideContactusInternal(),
    provideAppInfo({ appId: 'debtus', appTitle: 'Debtus.app' }),
    provideRouter([...appRoutes, ...authRoutes]),
    provideRolesByType(undefined),
  ],
}).catch((err) => console.error(err));

registerIonicons();
