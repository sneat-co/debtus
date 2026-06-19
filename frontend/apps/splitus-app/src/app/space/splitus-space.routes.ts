import { Route } from '@angular/router';
import { spacePagesRoutes } from '@sneat/ext-splitus-internal';
import {
  SpaceComponentBaseParams,
  SpaceMenuComponent,
} from '@sneat/space-components';

// Thin, splitus-only space shell. It provides SpaceComponentBaseParams (which
// resolves the active space from the :spaceType/:spaceID route params) to all
// children, then mounts ONLY the splitus routes — unlike sneat-app's
// @sneat/space-pages, which bundles every extension. This keeps splitus.app
// decoupled while reusing the published @sneat/space-components context wiring.
export const splitusSpaceRoutes: Route[] = [
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
