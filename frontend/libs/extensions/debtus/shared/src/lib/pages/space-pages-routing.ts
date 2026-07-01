import { Route } from '@angular/router';

// Debtus space-scoped routes, mounted under space/:spaceType/:spaceID.
// Full flow: debts (home/balances) -> debtus-contacts -> debtus-contact/:id
//   -> new-transfer / settle-up -> transfer/:id (receipt).
export const spacePagesRoutes: Route[] = [
  {
    path: 'debts',
    data: { title: 'Debts' },
    loadComponent: () =>
      import('./debtus-home/debtus-home-page.component').then(
        (m) => m.DebtusHomePageComponent,
      ),
  },
  {
    path: 'debtus-contacts',
    data: { title: 'Contacts' },
    loadComponent: () =>
      import('./debtus-contacts/debtus-contacts-page.component').then(
        (m) => m.DebtusContactsPageComponent,
      ),
  },
  {
    path: 'debtus-contact/:contactID',
    data: { title: 'Contact' },
    loadComponent: () =>
      import(
        './debtus-contact-details/debtus-contact-details-page.component'
      ).then((m) => m.DebtusContactDetailsPageComponent),
  },
  {
    path: 'new-transfer',
    data: { title: 'New transfer' },
    loadComponent: () =>
      import('./new-transfer/new-transfer-page.component').then(
        (m) => m.NewTransferPageComponent,
      ),
  },
  {
    path: 'settle-up',
    data: { title: 'Settle up' },
    loadComponent: () =>
      import('./settle-up/settle-up-page.component').then(
        (m) => m.SettleUpPageComponent,
      ),
  },
  {
    path: 'transfer/:transferID',
    data: { title: 'Transfer' },
    loadComponent: () =>
      import('./transfer-details/transfer-details-page.component').then(
        (m) => m.TransferDetailsPageComponent,
      ),
  },
];
