# debtus/backend — target scaffold structure

Blueprint for the repo after the Move phase. Module:
`github.com/sneat-co/debtus/backend` (rooted at `debtus/backend/`). Holds two
extensions (debtus + splitus), both bots, and reminders.

> These directories/files are created during the **Move phase**, not now — the
> live source is still in sneat-go. This doc is the blueprint to execute against.

## Proposed package tree

```
debtus/backend/
├── go.mod                       # module github.com/sneat-co/debtus/backend
├── cmd/
│   └── debtusd/main.go          # server assembly (grows from current health-only scaffold)
├── internal/
│   └── health/                  # existing scaffold (untouched)
├── config/                      # NEW: typed config; externalized secrets (P4)
│   └── config.go                #   Twilio/Apple/GA/OneSignal from env/secret-manager
│
├── debtus/                      # ── debtus extension ──
│   ├── module.go                #   Extension() (from sneat-go pkg/modules/debtus/module.go)
│   ├── const4debtus/  models4debtus/  dal4debtus/  debtusdal/
│   ├── facade4debtus/  general4debtus/  errors4debtus/
│   ├── api/{api4debtus, api4transfers}/
│   ├── analytics2debtus/  sms/  webhooks/  onesignal/  stats/
│   ├── trans4debtus/  utmconsts/  delayer4debtus/  apps/  debtusbotconst/
│   ├── common4debtus/           #   minus secrets.go (-> config/)
│   └── email4debtus/            #   NEW: receipt/invite email moved out of sneat-go pkg/modules/invites (P3)
│
├── splitus/                     # ── splitus extension ──
│   ├── module.go                #   Extension()
│   ├── const4splitus/  models4splitus/  briefs4splitus/
│   ├── dal4splitus/  facade4splitus/  api4splitus/  api4splitusbot/
│   └── ...
│
├── bots/                        # ── telegram bots (move WITH extensions) ──
│   ├── debtusbot/               #   from sneat-go pkg/bots/botprofiles/debtusbot
│   ├── splitusbot/              #   from sneat-go pkg/bots/botprofiles/splitusbot
│   └── delayers4debtusbot/      #   from sneat-go pkg/bots/delayers4debtusbot
│
└── reminders/                   # from sneat-go pkg/reminders (debtus-only)
    ├── dbo4reminders/  dal4reminders/  api4reminders/  delay4reminders/
    └── (remove the debtusdal.DelayerSendReminder circular-import workaround)
```

Depends on: `sneat-go-core`, `sneat-core-modules`, `sneat-bots` (P2), dalgo,
crediterra/{money,go-interest}, strongo/{gamp,gotwilio,delaying,i18n,…}. See
`MIGRATION-PLAN.md` for the require list (finalize after P1 stabilizes deps).

## Extension wiring skeletons

Based on the real current debtus `Extension()`. NOTE: `RegisterRoutes` /
`RegisterDelays` OVERWRITE on repeat — each must be passed once with a combined
closure (the live code has a TODO noting this silently dropped routes once).

```go
// debtus/module.go
package debtus

func Extension() extension.Config {
    debtusdal.RegisterDal()
    return extension.NewExtension(const4debtus.ModuleID,
        extension.RegisterRoutes(func(handle extension.HTTPHandleFunc) {
            handleWithContext := adaptHandler(handle)
            api4debtus.InitApiForDebtus(handleWithContext)
            api4transfers.InitApiForTransfers(handleWithContext)
        }),
        extension.RegisterDelays(func(reg func(key string, i any) delaying.Delayer) {
            facade4debtus.InitDelays4debtus(reg)
            debtusdal.RegisterDelayers4Debtus(reg)
            delayers4debtusbot.InitDelayers(reg)   // now .../debtus/backend/bots/delayers4debtusbot
            api4unsorted.InitDelaying(reg)          // now .../bots/debtusbot/api4unsorted
            reminders.InitDelaying(reg)             // now .../debtus/backend/reminders
        }),
    )
}
```

```go
// splitus/module.go
package splitus

func Extension() extension.Config {
    return extension.NewExtension(const4splitus.ModuleID,
        extension.RegisterRoutes(func(handle extension.HTTPHandleFunc) { /* api4splitus */ }),
        extension.RegisterDelays(func(reg ...) { /* splitus delays */ }),
    )
}
```

## Server assembly (cmd/debtusd)

The scaffold's main currently serves only `/health`. Two consumption modes:

1. **As a library (primary):** sneat-go imports this module and appends
   `debtus.Extension()` + `splitus.Extension()` to its module list, and registers
   `bots/debtusbot` + `bots/splitusbot` in its `botinit` registry. `cmd/debtusd`
   stays a thin standalone for local/health.
2. **Standalone server (future):** `cmd/debtusd` reuses the sneat server bootstrap
   (router + Firestore + Cloud Tasks + sneat-bots webhook) to run the two
   extensions independently. Defer until needed.

Recommended now: mode 1 — keep `cmd/debtusd` minimal; the app (sneat-go) remains
the runtime host that wires router/DB/delayer/bot-webhook.

## Bot registration

debtusbot/splitusbot build on `sneat-bots` (anybot base). They are registered by
the host app's `pkg/bots/botinit` registry (which stays in sneat-go). After the
move, botinit imports the profiles from this repo instead of in-tree. The
`cmds4invites` debtus-invite plugin stays in sneat-go (see P2 contamination note)
or moves to `debtus/email4debtus` — decide during P3.

## Open items
- Finalize go.mod require versions after P1.
- Confirm splitus subpackage names (briefs4splitus, dal4splitus, etc.) at move time.
- Decide cmds4invites final home (sneat-go plugin vs debtus/email4debtus).
