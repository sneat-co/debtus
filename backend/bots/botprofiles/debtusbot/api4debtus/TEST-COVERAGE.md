# TEST-COVERAGE.md — api4debtus

## Coverage metrics

| Run      | Coverage | Uncovered statements |
|----------|----------|----------------------|
| Pre-run  | 96.2%    | ~9 (3 functions)     |
| Post-run | 98.3%    | 4 (1 documented gap) |

## Seams added

In `api_receipt.go`:

- `getReceiptChannel` converted from a `func` declaration to an overridable
  package-level `var getReceiptChannel = func(...)`. This lets a test return an
  arbitrary (non-`errUnknownChannel`) error to cover the otherwise-unreachable
  `else` branch of `HandleSetReceiptChannel` (lines 289-292).

Existing seams reused (not added): `dal4debtus.Default.Receipt`,
`dal4debtus.Default.HttpClient`, `getTransferByID`, `getUserByID`,
`checkTransferCreatorName`, `getBotSettingsByProfile`, `sendReceiptByEmail`,
`renderReceiptTemplate`, `facade.GetSneatDB`, `dal4debtus.HttpAppHost`.

The `renderReceiptTemplate` default-seam body (the `RenderTemplate` call) is
covered by calling the var directly with a nil-map single-locale translator and
recovering from the panic that occurs inside template rendering (the target line
registers as covered before the panic).

## Documented gaps

### os/env-dependent (uncoverable without a production fix — bug)

**Function:** `HandleCreateReceipt`, lines 391-395 (the `case 5:` block of the
`Accept-Language` parser).

**Reason:** `localeCode5` is built as
`strings.ToLower(al[:2]) + "-" + strings.ToUpper(al[4:])`. For any well-formed
5-char locale token `"xx-YY"`, `al[4:]` yields only the LAST character (`"Y"`),
producing a 4-char string like `"xx-Y"`. `i18n.GetLocaleByCode5` PANICS on any
code not present in its locale table, and no 4-char code5 exists there, so the
`locale.Code5 != ""` success path can never be reached without triggering a
production panic. Any 5-char `Accept-Language` token therefore crashes the
handler rather than reaching the covered branch.

**Refactor required:** fix the slice expression to `al[3:]` (or parse the region
properly) so a valid 5-char code5 is produced, and/or guard
`GetLocaleByCode5` against unknown codes instead of relying on its panic. Both
are logic changes, out of scope for a seam-only coverage unit.
