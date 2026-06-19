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
      import('./home/splitus-home-page.component').then(
        (m) => m.SplitusHomePageComponent,
      ),
    canActivate: [AuthGuard],
    data: { authGuardPipe: () => redirectToLoginIfNotSignedIn },
  },
  {
    // Space-scoped routes host the splitus pages, mirroring sneat-app's
    // space/:spaceType/:spaceID mount point.
    path: 'space/:spaceType/:spaceID',
    loadChildren: () =>
      import('./space/splitus-space.routes').then((m) => m.splitusSpaceRoutes),
  },
  {
    // sneat-auth-menu-item navigates here on sign-out; mirror sneat-app and
    // redirect to the login page (where the sign-in form is shown).
    path: 'signed-out',
    pathMatch: 'full',
    redirectTo: 'login',
  },
  {
    // User profile (linked auth accounts, country). Linked from the side menu's
    // sneat-auth-menu-item "signed in as" row. Guarded like the home page.
    path: 'my',
    loadComponent: () =>
      import('./my/my-profile-page.component').then(
        (m) => m.MyProfilePageComponent,
      ),
    canActivate: [AuthGuard],
    data: { title: 'My profile', authGuardPipe: () => redirectToLoginIfNotSignedIn },
  },
];
