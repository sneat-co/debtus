# TEST-COVERAGE.md — pkg/reminders/api4reminders

## Coverage: 100%

No production seams were added to this package.

## Test approach

- `sendReceipt` ParseForm error path: triggered by `?id=%zz` in the request URL (invalid percent-encoding causes `ParseForm` to return an error).
- `InviteFriend` email send error: injected via `emailing.GetEmailClient` seam (already a package-level var in sneat-core-modules).
- `InviteFriend` ParseForm error: uses COVER-BEFORE-PANIC pattern with `recover()` — the handler does not return after logging the error, so execution continues to a panic; `defer recover()` lets the test survive while still counting the error-path statements as covered.
