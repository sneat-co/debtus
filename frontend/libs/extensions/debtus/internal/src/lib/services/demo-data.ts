import {
  IContactBalance,
  IDebtusTransfer,
} from '@sneat/extension-debtus-contract';

// ===========================================================================
// Fable: prototype demo data
// ---------------------------------------------------------------------------
// These fixtures back the debtus web-UI flows that do NOT yet have a wired,
// authenticated Go HTTP endpoint (balances summary, contacts list, per-contact
// balance, transfer history). They are shaped exactly like the real contract
// models so that when the backend endpoints are wired (see PR notes:
// api4unsorted contacts CRUD + a balances endpoint), the internal service can
// swap `of(demo…)` for a `sneatApiService.get(…)` call with no UI changes.
//
// Counterparties are modelled as contactus space contacts (contactID = a
// contactus contact id) — debtus does NOT own a separate contact list.
// `bff-friend` is intentionally in a DIFFERENT space to exercise cross-space
// lending in the UI.
// ===========================================================================

export const DEMO_CONTACT_BALANCES: readonly IContactBalance[] = [
  {
    contactID: 'contact-alice',
    title: 'Alice Johnson',
    balance: { USD: 120, EUR: 0 },
  },
  {
    contactID: 'contact-bob',
    title: 'Bob Smith',
    balance: { USD: -45 },
  },
  {
    contactID: 'contact-carol',
    title: 'Carol Lee',
    balance: { EUR: 30 },
  },
  {
    // Cross-space counterparty: belongs to their own personal space.
    contactID: 'contact-dave',
    title: 'Dave (family friend)',
    balance: { USD: 200 },
    counterpartySpaceID: 'space-dave-personal',
  },
  {
    contactID: 'contact-erin',
    title: 'Erin Park',
    balance: { EUR: 0, USD: 0 }, // settled up
  },
];

export function demoTransfersForSpace(spaceID: string): IDebtusTransfer[] {
  return [
    {
      id: 'transfer-1001',
      direction: 'lend',
      amount: { currency: 'USD', value: 120 },
      counterpartyContactID: 'contact-alice',
      counterpartyTitle: 'Alice Johnson',
      note: 'Concert tickets',
      created: '2026-06-20T10:15:00Z',
      dueOn: '2026-07-20T00:00:00Z',
      isReturn: false,
      isOutstanding: true,
      creatorSpaceID: spaceID,
    },
    {
      id: 'transfer-1002',
      direction: 'borrow',
      amount: { currency: 'USD', value: 45 },
      counterpartyContactID: 'contact-bob',
      counterpartyTitle: 'Bob Smith',
      note: 'Lunch',
      created: '2026-06-22T12:30:00Z',
      isReturn: false,
      isOutstanding: true,
      creatorSpaceID: spaceID,
    },
    {
      id: 'transfer-1003',
      direction: 'lend',
      amount: { currency: 'EUR', value: 30 },
      counterpartyContactID: 'contact-carol',
      counterpartyTitle: 'Carol Lee',
      created: '2026-06-25T09:00:00Z',
      isReturn: false,
      isOutstanding: true,
      creatorSpaceID: spaceID,
    },
    {
      id: 'transfer-1004',
      direction: 'lend',
      amount: { currency: 'USD', value: 200 },
      counterpartyContactID: 'contact-dave',
      counterpartyTitle: 'Dave (family friend)',
      note: 'Cross-space loan (different space)',
      created: '2026-06-18T08:00:00Z',
      isReturn: false,
      isOutstanding: true,
      creatorSpaceID: spaceID,
      counterpartySpaceID: 'space-dave-personal',
    },
    {
      id: 'transfer-1005',
      direction: 'borrow',
      amount: { currency: 'USD', value: 20 },
      counterpartyContactID: 'contact-erin',
      counterpartyTitle: 'Erin Park',
      note: 'Coffee (already returned)',
      created: '2026-06-10T08:00:00Z',
      isReturn: false,
      isOutstanding: false,
      creatorSpaceID: spaceID,
    },
  ];
}
