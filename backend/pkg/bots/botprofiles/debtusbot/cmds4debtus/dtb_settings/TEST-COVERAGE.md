# Test Coverage

Package: `github.com/sneat-co/sneat-go/pkg/bots/botprofiles/debtusbot/cmds4debtus/dtb_settings`

## Coverage metrics

| Metric | Value |
|--------|-------|
| Pre-run coverage | 89.9% |
| Post-run coverage | 98.0% |
| Uncovered statements remaining | 3 |

## Seams added

None added in this run. The package already exposed the package-level seam vars
needed (`getUser`, `runReadwriteTransaction`, `runUserWorker`, `getInvite`,
`getReceiptByID`, `claimInvite2`, `delaySetUserPreferredLocale`, `showReceipt`,
`acknowledgeReceipt`, `setMainMenuKeyboard`, `settingsMainAction`, `assignPinCode`).

## How the previously-uncovered closures were covered

- `fixbalance_cmd.go` transaction worker: the `runReadwriteTransaction` test stub
  now INVOKES the worker callback against an in-memory DB
  (`sneattesting.SetupMemoryDB`) instead of returning nil, so the callback body
  (build space entry, `tx.Set`) is exercised. A second test stubs the seam to
  return an error to cover the `if err = runReadwriteTransaction(...); err != nil`
  branch.
- `currency_cmd.go` `SetPrimaryCurrency` worker: the `runUserWorker` test stub now
  invokes the worker callback with a `*dal4userus.UserWorkerParams{User: ...}`,
  covering `userWorkerParams.User.Data.SetPrimaryCurrency(...)`.
- Default seam closures (`getInvite`, `getReceiptByID`, `claimInvite2`):
  `TestDefaultSeamClosures` sets `dal4debtus.Default.Invite` / `.Receipt` to fakes
  that embed the interface and override only the needed method, then calls the
  closures directly.

## Documented gaps

### dead

#### `fixbalance_cmd.go:34` — `if err != nil { return err }` inside the tx worker

- **whyType**: dead
- **reason**: `err` here is the function's outer named return value, which is
  guaranteed nil at this point (the preceding `getUser` error already returned).
  The branch is therefore unreachable.
- **refactorRequired**: Remove the dead `if err != nil` check (forbidden under the
  seam-only rule; documented as a gap instead).

#### `fixbalance_cmd.go:38-41` — `for _, contact := range debtusSpace.Data.Contacts` loop body

- **whyType**: dead
- **reason**: `debtusSpace` is built fresh via `NewDebtusSpaceEntry(spaceID)` inside
  the worker and is never loaded from the DB (`tx.Get` is never called), so
  `debtusSpace.Data.Contacts` is always empty and the loop body never runs. This
  is a latent production bug (the balance recompute reads from a record that was
  never populated).
- **refactorRequired**: Load the existing debtus-space record (`tx.Get`) before the
  loop. Logic change — out of scope for seam-only coverage work.
