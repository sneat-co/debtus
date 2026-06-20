import { Route } from '@angular/router';
import { spacePagesRoutes } from '@sneat/extension-debtus-shared';
import {
  SpaceComponentBaseParams,
  SpaceMenuComponent,
} from '@sneat/space-components';

// Thin, debtus-only space shell. It provides SpaceComponentBaseParams (which
// resolves the active space from the :spaceType/:spaceID route params) to all
// children, then mounts ONLY the debtus routes — unlike sneat-app's
// @sneat/space-pages, which bundles every extension. This keeps debtus.app
// decoupled while reusing the published @sneat/space-components context wiring.
export const debtusSpaceRoutes: Route[] = [
  {
    path: '',
    providers: [SpaceComponentBaseParams],
    children: [
      {
        path: '',
        component: SpaceMenuComponent,
        outlet: 'menu',
      },
      {
        path: '',
        pathMatch: 'full',
        redirectTo: 'debts',
      },
      ...spacePagesRoutes,
    ],
  },
];
