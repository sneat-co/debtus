# P2: Extract `github.com/sneat-co/sneat-bots` (shared bot framework)

Prerequisite for moving debtusbot/splitusbot (and listusbot) with their
extensions. Depends on P1 (dalgo v0.62.2) — anybot uses `record.DataWithID`.

## What moves to sneat-bots (generic, ~93 .go files incl. tests)
From `sneat-go/pkg/bots`:
- `botprofiles/anybot` (NewProfile, SneatBotBaseData, SneatAppTgChatDbo/Entry, SneatAppTgUserDbo)
- `botprofiles/anybot/cmds4anybot` **minus `cmds4invites`** (see contamination)
- `botprofiles/anybot/const4anybot`
- `botprofiles/anybot/facade4anybot` (after cleaning debtus errors)
- `bothelper`, `botsettings`, `botinitparams`
- `sneatbots/{facade4bots, models4bots, tghelpers}` + `sneatbots` root (BotTokensProvider)

## Stays in sneat-go
- `bots/botinit` (hardcoded bot registry, app wiring)
- `bots/botauth` (Firebase user create/delete)
- all `botprofiles/<app>bot` (debtusbot, splitusbot, listusbot… — these move to
  their own extension repos, not to sneat-bots)

## ⚠️ App-specific contamination to clean BEFORE/DURING extraction
The "generic" anybot currently leaks app dependencies — these must be removed so
sneat-bots depends on NOTHING app-specific:

| Location | Bad dependency | Fix |
|---|---|---|
| `anybot/cmds4anybot/cmds4invites/*` | `modules/debtus/{dal4debtus,models4debtus}`, `modules/invites`, `debtusbot/.../dtb_general` | Do NOT extract. Keep in sneat-go as an app-level invite plugin (ties into invites consolidation, issue sneat-go#672). |
| `anybot/facade4anybot/auth_facade.go` | `modules/debtus/errors4debtus` | Replace with a generic error type defined in sneat-bots. |
| `anybot/cmds4anybot/start_command.go` | hardcoded `https://debtus.app/...` URLs | Parameterize via `BotParams` (per-profile config). |
| `anybot/cmds4anybot` | `modules/togethered/const4togd` | Parameterize or move the constant into sneat-bots config. |

Net: sneat-bots must end with zero imports of `sneat-go/pkg/modules/*`,
`bots/botinit`, `bots/botauth`.

## Dependency surface (sneat-bots go.mod)
```
module github.com/sneat-co/sneat-bots
go 1.26
require (
  github.com/bots-go-framework/bots-api-telegram v0.14.8
  github.com/bots-go-framework/bots-fw v0.71.56
  github.com/bots-go-framework/bots-fw-store v0.10.3
  github.com/bots-go-framework/bots-fw-telegram v0.25.39
  github.com/bots-go-framework/bots-fw-telegram-models v0.3.57
  github.com/dal-go/dalgo v0.62.2
  github.com/sneat-co/sneat-go-core v0.55.2
  github.com/sneat-co/sneat-core-modules v0.38.55
  github.com/sneat-co/sneat-translations v0.7.108
  github.com/strongo/i18n v0.8.10
  github.com/strongo/delaying v0.2.1
  github.com/strongo/logus v0.4.1
)
```
(Verify versions against sneat-go/go.mod at execution time.)

## Reverse deps (must repoint after extraction)
ALL bot profiles: anybot, debtusbot, splitusbot, listusbot, trackusbot, sneatbot,
assetusbot, datatugbot, rosycyclebot, togetheredbot, collectusbot.

## Steps
1. Create repo `github.com/sneat-co/sneat-bots` (go.mod above).
2. Copy the generic package set; rewrite imports `sneat-go/pkg/bots/...` ->
   `sneat-bots/pkg/bots/...`.
3. Clean the 4 contamination points above.
4. `go build ./... && go test ./...` green in sneat-bots.
5. In sneat-go: add require + local replace (=> ../sneat-bots); repoint all bot
   profiles + botinit/botsettings consumers; keep `cmds4invites` locally.
6. `go build ./... && go test ./...` green in sneat-go.

## Risk
Low-to-medium: mostly mechanical import repoints, but the 4 contamination fixes
require small refactors (generic error type, BotParams URL config). No behavior
change intended.
