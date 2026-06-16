# Test Coverage Notes

Package: `github.com/sneat-co/sneat-go/pkg/bots/botprofiles/debtusbot/cmds4debtus/dtb_general`

## Coverage metrics

- Pre-run coverage: **98.9% of statements** (3 uncovered statements)
- Post-run coverage: **99.6% of statements** (1 uncovered statement)

## Seams added

None added in this run. The package already had package-level seam vars
(`deleteAll`, `getFeedbackByID`, `runReadwriteTransaction`, `getCurrentSpaceRef`,
`getMessageUID`, `textReceiptForTransfer`, ...). The two default seam-closure
bodies (`deleteAll`, `getFeedbackByID`) are now covered by `TestDefaultSeamClosures`,
which sets `dal4debtus.Default.Admin` / `dal4debtus.Default.Feedback` to fake
implementations (embedding the interface) and calls the closures directly.

## Documented Gaps

### dead

#### `delete_all_cmd.go:28-30` — `else if Env == "prod"` dead branch

**Why uncoverable:** Structurally unreachable. The outer `if` on line 26 returns
for any `Env != LocalHostEnv && Env != "dev"`. `"prod"` satisfies that, so the
`else if botSettings.Env == "prod"` on line 28 can never be reached.

**Refactor required:** None for coverage; delete the dead branch in a separate
cleanup PR (seam-only rule forbids changing it here).
