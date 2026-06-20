// Main entry point for splitus.app
import { bootstrapApplication } from '@angular/platform-browser';
import { provideRouter } from '@angular/router';
import {
  getStandardSneatProviders,
  provideAppInfo,
  provideRolesByType,
} from '@sneat/app';
import { authRoutes } from '@sneat/auth-ui';
import { provideSplitusInternal } from '@sneat/extension-splitus-internal';
import { App } from './app/app';
import { appRoutes } from './app/app.routes';
import { splitusAppEnvironmentConfig } from './environments/environment';
import { registerIonicons } from './register-ionicons';

bootstrapApplication(App, {
  providers: [
    ...getStandardSneatProviders(splitusAppEnvironmentConfig),
    // Bind the splitus contract tokens (SPLITUS_SERVICE) to their concrete
    // implementations. The app is the composition root and may wire -internal.
    ...provideSplitusInternal(),
    provideAppInfo({ appId: 'splitus', appTitle: 'Splitus.app' }),
    provideRouter([...appRoutes, ...authRoutes]),
    provideRolesByType(undefined),
  ],
}).catch((err) => console.error(err));

registerIonicons();
