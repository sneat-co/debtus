import { Route } from '@angular/router';

// Splitus space-scoped routes, mounted under space/:spaceType/:spaceID.
// Flow: splits (home) --"New split"--> new-split --create--> split/:splitID.
export const spacePagesRoutes: Route[] = [
  {
    path: 'splits',
    data: { title: 'Splits' },
    loadComponent: () =>
      import('./splitus-home/splitus-home-page.component').then(
        (m) => m.SplitusHomePageComponent,
      ),
  },
  {
    path: 'new-split',
    data: { title: 'New split' },
    loadComponent: () =>
      import('./new-split/new-split-page.component').then(
        (m) => m.NewSplitPageComponent,
      ),
  },
  {
    path: 'split/:splitID',
    data: { title: 'Split' },
    loadComponent: () =>
      import('./split-details/split-details-page.component').then(
        (m) => m.SplitDetailsPageComponent,
      ),
  },
];
