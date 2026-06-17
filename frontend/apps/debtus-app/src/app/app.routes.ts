import { Route } from '@angular/router';
import { AuthGuard } from '@angular/fire/auth-guard';
import { redirectToLoginIfNotSignedIn } from '@sneat/auth-core';

export const appRoutes: Route[] = [
  {
    // Authenticated landing: lists the user's spaces. Unauthenticated visitors
    // are redirected to /login by the auth guard. Replaces the previous
    // redirectTo:'login', which bounced signed-in users back to the login page.
    path: '',
    pathMatch: 'full',
    loadComponent: () =>
      import('./home/debtus-home-page.component').then(
        (m) => m.DebtusHomePageComponent,
      ),
    canActivate: [AuthGuard],
    data: { authGuardPipe: () => redirectToLoginIfNotSignedIn },
  },
  {
    // Space-scoped routes host the debtus pages, mirroring sneat-app's
    // space/:spaceType/:spaceID mount point.
    path: 'space/:spaceType/:spaceID',
    loadChildren: () =>
      import('./space/debtus-space.routes').then((m) => m.debtusSpaceRoutes),
  },
];
