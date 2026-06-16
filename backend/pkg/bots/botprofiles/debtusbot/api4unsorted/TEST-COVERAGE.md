# TEST-COVERAGE

Package: `github.com/sneat-co/sneat-go/pkg/bots/botprofiles/debtusbot/api4unsorted`

## Coverage metrics

Pre-run (this unit): **86.9%** | Post-run: **91.8%** | Uncovered statements remaining: **~50**
(of which ~21 are seam-delegate default bodies in `seams.go` + `api_tg_helpers.go`,
intentionally untested by the seam pattern, and 0 belong to `HandlerDeleteGroup`
which has an empty body and therefore no countable statements).

## Seams added

- `sendToTelegramFn` (`api_tg_helpers.go`): `var sendToTelegramFn = sendToTelegram`.
  The goroutine callback in `HandleTgHelperCurrencySelected` now calls
  `sendToTelegramFn` instead of `sendToTelegram` directly, so the callback closure
  (botID construction + the call) can be covered by stubbing the seam, without
  performing real Telegram IO. Default body is the real `sendToTelegram`.

No other production changes were made.

## Newly covered this run

- `DelayedChangeTransfersCounterparty`: now **100%**. Added `errRecordsReader`
  (a `dal.RecordsReader` whose `Next()` returns a non-`ErrNoMoreRecords` error) to
  drive the `dal.SelectAllIDs` error path (`api_admin.go:124-126`).
- `HandleAdminMergeUserContacts`: now **97.4%**. Added `errDelayer` (a
  `delaying.Delayer` whose `EnqueueWork` returns an error) to cover the
  `EnqueueWork` error branch (`api_admin.go:89-91`). `delayChangeTransfersCounterparty`
  is a package-level var (assigned by `InitDelaying`) so it is swappable in tests.
- `HandleTgHelperCurrencySelected`: now **94.2%**. The goroutine callback closure
  (`api_tg_helpers.go:95-96`) is covered via a `callbackTgChatDal` fake that invokes
  the callback, plus the new `sendToTelegramFn` seam.
- `sendToTelegram`: now **18.2%** (was 0%). The `GetBotSettingsByCode` error path
  (`api_tg_helpers.go:115-119`) is covered by calling `sendToTelegram` directly with
  an unknown bot code (`botSettingsProvider` is nil in tests).

## Documented gaps

### whyType: seam-delegate-body

Every `var f = func(...)` default body in `seams.go` (19 vars) is intentionally not
covered: in tests the seam var is always replaced with a fake, so the delegate body
that calls the real external dependency is never executed. Covering them would require
live DB / facade infrastructure (integration tests). Same applies to the default body
of `sendToTelegramFn` (it points at `sendToTelegram`, exercised partially by the
direct test above).

### whyType: dead-code-logical

`api_admin.go:74-76`: condition
`contactToDelete.Data.UserID != "" && contactToKeep.Data.UserID == ""` can never be
true because line 71 already returned if the two UserIDs differ — if they are equal,
they cannot simultaneously satisfy `!= ""` and `== ""`. Refactor required: remove the
redundant guard or reorder the checks. Not coverable under the seam-only rule.

### whyType: concurrency (panic-recovery defers)

`api_tg_helpers.go:73-75` and `88-90`: NOW COVERED this run. The two `recover()` log
lines are exercised by injecting a panicking `setLastCurrency` seam
(`TestHandleTgHelperCurrencySelected_SetLastCurrencyPanic`) and a panicking
`bots.TgChat.DoSomething` fake (`TestHandleTgHelperCurrencySelected_DoSomethingPanic`).
Because a panicked goroutine never sends the second value to the handler's `errs`
channel, the handler blocks; the tests run it in a goroutine bounded by a 2s timeout
(the recover line is recorded as covered regardless of whether the handler returns).

`api_tg_helpers.go:79-81`: `if err2 := setLastCurrency(...); err != nil` — the
condition checks the outer `err` (nil here) instead of `err2`. The error-log branch is
unreachable by design defect. Fixing requires correcting `err` to `err2` (a logic
change, outside the seam-only rule).

### whyType: external-io + panic (sendToTelegram body)

`api_tg_helpers.go:121-160` (the rest of `sendToTelegram`): after the
`GetBotSettingsByCode` success path it constructs a real `tgbotapi` client, calls
`NewApiWebhookContext` (which panics on empty `BotSettings.Code` — see below), and
calls `tgBotApi.Send` (real Telegram HTTP). Covering it needs a configured bot-settings
provider, a non-panicking `NewApiWebhookContext`, and a mock Telegram server —
integration-scope, beyond the seam-only rule.

### whyType: refactor-required (panic on empty BotSettings.Code)

`api_whc.go:40-67` (`NewApiWebhookContext` body after `NewBotContext`): the locally
declared `botSettings` has an empty `Code`, causing `botsfw.NewBotContext` to panic
before the rest of the function runs. Covering lines 40-67 requires giving
`botSettings.Code` a non-empty value — i.e. refactoring `NewApiWebhookContext` to
accept a `BotSettings` parameter or read it from a package-level var. Outside the
seam-only rule.

`api_contacts.go:276-279` (`HandleUpdateCounterparty` success path): calls
`contactToResponse` with a locally constructed `ContactEntry` whose `Names` is nil →
nil-pointer dereference in `Names.GetFullName()`. The `updateContact` seam returns
`DebtusSpaceContactEntry` only, with no way to populate `Names` without a production
change (add nil-guard or seam). Outside the seam-only rule.

`api_groups.go:72-76` (`groupToResponse` write path): `groupsToJson` always returns
`errors.New("groupsToJson not implemented yet")`, making the `w.Write` branch
structurally unreachable until `groupsToJson` is implemented.

`api_groups.go:216` (`HandlerDeleteGroup`): empty body — a route stub that does
nothing. Go emits no countable statement, so it cannot be brought above 0% without
implementing the handler.
