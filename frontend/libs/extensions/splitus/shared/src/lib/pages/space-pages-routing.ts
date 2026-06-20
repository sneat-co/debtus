import { Route } from '@angular/router';

export const spacePagesRoutes: Route[] = [
  {
    path: 'splits',
    data: { title: 'Splits' },
    loadComponent: () =>
      import('./splitus-home/splitus-home-page.component').then(
        (m) => m.SplitusHomePageComponent,
      ),
  },
];
