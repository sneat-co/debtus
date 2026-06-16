# TEST-COVERAGE.md — api4debtus

## Coverage metrics

| Run       | Coverage | Uncovered statements |
|-----------|----------|----------------------|
| Pre-run   | 97.5%    | 3                    |
| Post-run  | 98.3%    | 1 (documented)       |

## Seams added

In `api_receipt.go`, the following direct calls were replaced with overridable
package-level `var` seams so tests can avoid real DB / email / network I/O:

- `getTransferByIDFn = facade4debtus.Transfers.GetTransferByID`
- `checkTransferCreatorNameFn = facade4debtus.CheckTransferCreatorNameAndFixIfNeeded`
- `sendReceiptByEmailFn = invites.SendReceiptByEmail`
- `renderReceiptTemplateFn = func(...) { common4all.TextTemplates.RenderTemplate(...) }`
- `getReceiptChannel = func(r *http.Request) (channel string, err error) { ... }`
  (was a plain func; converted to a package-level `var` so tests can override it
  to return a non-`errUnknownChannel` error, covering the
  `StatusInternalServerError` else-branch of `HandleSetReceiptChannel`)

Existing seams reused (not added): `dal4debtus.Default.Receipt`,
`dal4debtus.Default.HttpClient`, `dal4userus.GetUserByID`, `facade.GetSneatDB`,
`facade2bots.GetBotID`, `dal4debtus.HttpAppHost`. The Google-Analytics flush in
`analytics2debtus.ReceiptSentFromApi` is neutralized by stubbing the existing
`dal4debtus.Default.HttpClient` field with a fake `http.RoundTripper`.

## Documented gaps

### bug (uncoverable without production fix)

**Function:** `HandleCreateReceipt`, lines 391-393 (the `case 5` Accept-Language
branch).

**Reason:** The production slicing `al[:2] + "-" + al[4:]` turns a 5-char code
like "en-US" into "en-S", which makes `i18n.GetLocaleByCode5` panic. No 5-char
Accept-Language value reaches `lang = al; goto langSet` without panicking, so
the branch is uncoverable until the slicing is fixed to `al[:2] + "-" + al[3:5]`.

**Refactor required:** fix the slicing bug (a logic change, not a seam — out of
scope). Captured as an observation.
