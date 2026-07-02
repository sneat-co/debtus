import { Component } from '@angular/core';

// Persistent "demo data" banner for fixture-backed READ surfaces.
//
// Balances, contacts and transfer history are still served from demo fixtures
// (no wired read endpoints yet — see DebtusService), while creates/settles
// WRITE real records. Without this banner users see fiction presented as
// their data, and the transfers they record never appear in any list. Show
// this on every page that renders fixture-backed reads; remove per page once
// its live read endpoint is wired.
@Component({
  selector: 'sneat-debtus-demo-data-banner',
  template: `
    <div class="demo-banner" role="note">
      ⚠️ <b>Demo data</b> — balances, contacts &amp; history shown here are
      sample fixtures, not your records yet. Transfers you record are saved for
      real and will show up once live reads are wired.
    </div>
  `,
  styles: [
    `
      .demo-banner {
        margin: 8px 12px 0;
        padding: 8px 12px;
        border-radius: 8px;
        font-size: 0.85rem;
        background: var(--ion-color-warning-tint, #ffd534);
        color: var(--ion-color-warning-contrast, #000);
      }
    `,
  ],
})
export class DemoDataBannerComponent {}
