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
  {
    // sneat-auth-menu-item navigates here on sign-out; mirror sneat-app and
    // redirect to the login page (where the sign-in form is shown).
    path: 'signed-out',
    pathMatch: 'full',
    redirectTo: 'login',
  },
  {
    // sneat-auth-menu-item links the "signed in as" row to /my. Until debtus has
    // a real profile page, send it to the home landing. TODO: scaffold a profile.
    path: 'my',
    pathMatch: 'full',
    redirectTo: '',
  },
];
