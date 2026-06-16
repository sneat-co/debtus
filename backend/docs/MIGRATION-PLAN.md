# debtus (+ splitus) backend extraction plan

Extract the **debtus** and **splitus** modules — one bounded context — out of
`sneat-go` into this repo (`github.com/sneat-co/debtus/backend`, scaffolded with
`cmd/debtusd` + `internal/health`).

**Greenfield:** no backward compatibility. Legacy code may be deleted outright;
no import-path preservation or deprecation periods.

## What moves into this repo
- `sneat-go/pkg/modules/debtus/**` (185 files)
- `sneat-go/pkg/modules/splitus/**`
- `sneat-go/pkg/bots/botprofiles/debtusbot/**` + `pkg/bots/delayers4debtusbot`
- `sneat-go/pkg/bots/botprofiles/splitusbot/**`
- `sneat-go/pkg/reminders/**` (debtus-only; -> `debtus/backend/reminders`)
- debtus-specific email code currently in `sneat-go/pkg/modules/invites`
  (`SendReceiptByEmail` + debtus UTM) -> `debtus/backend/.../email4debtus`

## Decisions locked
- debtus + splitus + both bots + reminders live **together** in one repo
  (bidirectional coupling stays internal; no cycle to break between them).
  Future split tracked in `IDEA-split-debtus-and-splitus.md`.
- Bots move **with** their extension.
- Shared bot framework goes to a **new `github.com/sneat-co/sneat-bots` repo**.
- Generic reminders: deferred (idea in sneat-go `docs/REMINDERS-DECOUPLING-IDEA.md`).
- Generic invitations: deferred to a dedicated effort (issues sneat-go#672,
  sneat-core-modules#106); only the debtus-specific email moves now.

## Prerequisites (do in sneat-go first; each independently verifiable)

### P1. dalgo v0.62.2 migration across sneat-go  ← also unblocks listus  [DONE]
The local `sneat-go-core`/`sneat-core-modules` already require dalgo v0.62.2,
which replaced the `WithID` embed with `RecordWithID` (and changed
`dal.CollectionAt[T]` -> `CollectionAt[K,T]`). sneat-go migrated; all repos green.
PRs: listus#2, sneat-go-backend#192, sneat-go#673 (branch fix/dalgo-v0.62-migration).
Note: only `DataWithID`-alias types renamed to `RecordWithID`; structs embedding
`record.WithID` directly keep `WithID`.

### P2. Extract `sneat-bots` shared framework (new repo)
Move shared bot scaffolding out of `sneat-go/pkg/bots` into
`github.com/sneat-co/sneat-bots`. See `P2-SNEAT-BOTS-EXTRACTION-PLAN.md`.
(Depends on P1 — anybot uses the new dalgo API.)

### P3. Break debtus <-> invites (debtus half only)
Move `SendReceiptByEmail` + debtus-specific `SendInviteByEmail` bits from
`sneat-go/pkg/modules/invites/send_by_email.go` into debtus (`email4debtus`).
Repoint debtus + debtusbot call sites. Generic remainder tracked by sneat-go#672.

### P4. Externalize debtus secrets
See `P4-SECRETS-EXTERNALIZATION.md`. Rotate live Twilio/Apple creds.

### P5. Move reminders under debtus
`sneat-go/pkg/reminders` -> `pkg/modules/debtus/reminders` (in place first),
removing the `debtusdal.DelayerSendReminder` circular-import workaround. Verify.

## Move phase
1. Copy debtus + splitus + both bots + reminders + email4debtus into
   `debtus/backend` (see `SCAFFOLD-STRUCTURE.md`).
2. `go.mod`: require sneat-go-core, sneat-core-modules, sneat-bots, dalgo,
   strongo, crediterra/{money,go-interest}, gotwilio, gamp, delaying, etc. Add
   local `replace` directives to sibling checkouts for dev.
3. Rewrite imports `sneat-go/pkg/modules/debtus|splitus`, `sneat-go/pkg/bots/...`,
   `sneat-go/pkg/reminders` -> `debtus/backend/...`.
4. Wire `Extension()` entrypoints (debtus + splitus) and bot registration.
5. `go build ./... && go test ./...` green in debtus/backend.

## sneat-go re-wiring (consumer)
1. Delete the moved packages from sneat-go.
2. Add `require` + local `replace` (=> ../debtus/backend).
3. Register debtus/splitus extensions from the app (`pkg/sneatmain`); register
   the bots in `pkg/bots/botinit`.
4. Repoint remaining one-way reverse deps into the new repo:
   `pkg/bots/botauth/api4botauth` (errors4debtus), `anybot/cmds4invites`
   (debtus invite send), any others surfaced by the build.
5. `go build ./... && go test ./...` green in sneat-go.

## Open items to resolve during execution
- Enumerate every sneat-go reverse dependency on splitus (not just debtus) that
  must stay in sneat-go and become one-way into the new repo.
- Confirm splitusbot has no sneat-go-only dependencies beyond sneat-bots.

## Verification
- `go build ./... && go test ./...` green in: sneat-bots, debtus/backend,
  sneat-go-backend, sneat-go.
- No references to moved package paths remain in sneat-go.
