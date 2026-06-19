# Test Coverage

## Coverage metrics

| Metric | Value |
|--------|-------|
| Pre-run coverage | 7.5% |
| Post-run coverage | 11.5% |
| Uncovered statements remaining | 1601 of 1809 |

This package is dominated by bot-command / callback handlers that glue together a
live `botsfw.WebhookContext`, multiple DAL round-trips (load transfer, load receipt,
load contacts), Telegram/email side effects and the multi-step wizard state machine.
Those functions cannot be exercised under the seam-only production rule; they need a
full in-memory bot-framework + DAL integration harness that does not exist in this
repo yet. This run covered every directly-testable pure / decision function called
out in the researcher annotations.

## Functions covered in this run

- `SendReceiptCallbackData`, `SendReceiptUrl` (pure fmt wrappers)
- `IsCurrencyIcon` (pure lookup, both branches)
- `cleanPhoneNumber` (pure transform)
- `GetUrlForReceiptInTelegram` (pure URL builder)
- `getReturnDirectionFromDebtValue` (all branches incl. zero-value error)
- `IsTransferNotificationsBlockedForChannel` (pure — iterates an in-memory slice,
  NOT DAL-backed as earlier notes claimed)
- `shortDate` (en-US and default-locale branches)
- `BalanceForCounterpartyWithHeader` (pure formatter)
- `GetTransferSource` (via mock_botsfw WebhookContext)
- `callbackTransferHistory` (TODO message)
- `TransferWizardCompletedCommand` unknown-code error branch
- `BalanceMessageBuilder.ByContact` error-row + zero-balance skip paths (via seam)
- `getInterestData` (pure slash-separated parser — valid + all 5 error branches)
- `TransferWizard.CounterpartyID` (pure — counterparty key, contact-key fallback, empty)
- `sendReceiptByTelegramButton` (pure switch-inline-query button builder)
- `_debtAmountButtonText` (all 3 switch arms via mock `whc.Translate`)
- `getReturnWizardParams` (well-formed + malformed-query error branch via mock `whc.ChatData`)
- `AskTransferCurrencyButtons` (no-last-currencies + with-last-currencies, via a fake
  `botsfwmodels.AppUserData` implementing `GetLastCurrencies()`)

## Seams added

- **transfer_balance.go** — `var balanceWithInterestFn = func(o *models4debtus.DebtusContactBrief, ctx context.Context, now time.Time) (money.Balance, error) { return o.BalanceWithInterest(ctx, now) }`.
  Call site at line 79 swapped from `debtusContactBrief.BalanceWithInterest(ctx, now)`
  to `balanceWithInterestFn(debtusContactBrief, ctx, now)`. Required because
  `DebtusContactBrief.BalanceWithInterest` calls `updateBalanceWithInterest` with
  `failOnZeroBalance=false`, making the error branch in `ByContact` (lines 80-84)
  unreachable through data alone. The seam lets a test inject an error.

## Documented gaps

### defensive/unreachable

- **`DurationToString` inner `case 1:` (one minute) — due_returns_cmd.go:128**
  `d.Hours()==0` only when `d==0`, which also makes `d.Minutes()==0`, so `d.Minutes()`
  can never equal exactly 1 inside the `case 0:` hours branch. The function is 80%
  covered (all reachable arms).
  Required refactor (NOT a seam, so NOT applied): change `switch hours { case 0: ... }`
  to `if hours < 1.0 { switch { ... } }` so sub-hour non-zero durations reach the
  minute arms.

### external-io (webhook + DAL integration required)

Bot-command/callback Action closures and their helpers. Each requires a live (or fully
faked) `botsfw.WebhookContext` supplying `Input()` as `botinput.TextMessage`, chat-data
wizard state, AND one or more DAL reads/writes (`facade4debtus.Transfers.*`,
`dal4debtus.*`, user/contact workers) plus Telegram/email side effects. Covering them
needs an in-memory bot+DAL integration harness; under the seam-only rule they are gaps:

- `sendReceiptCallbackAction`, `showLinkForReceiptInTelegram`, `sendReceiptBySms`,
  `sendReceiptByEmail`, `AcknowledgeReceipt`, `ShowReceipt`, `viewReceiptCallbackAction`
- `dueReturnsCallbackAction`, `showReceiptAnnouncement`, `OnInlineChosenCreateReceipt`,
  `InlineSendReceipt`, `getInlineReceiptMessageText`, `newCounterpartyCommand`
- `ProcessReturnAnswer`, `ProcessFullReturn`, `processNoReturn`, `askWhenToRemindAgain`
- `processSetDate` (transfer_create.go) — parses `whc.Input().(botinput.TextMessage)`;
  would be trivially unit-testable if the date-parsing core were extracted into a pure
  `parseTransferDate(text string) (time.Time, error)` helper (refactor not allowed here).
- `ProcessPartialReturn` deeper paths beyond the entry guard — needs TextMessage input +
  DAL-backed transfer; only the unrelated-user guard is reachable without the harness
  (covered by an existing test).
- The bulk of `transfer_common.go`, `transfer_create.go`, `transfer_return.go`,
  `transfer_history.go`, `receipt.go`, `invite.go`, `new_contact_cmd.go`,
  `ack_receipt_cmd.go`, `ask_email_cmd.go`, `receipt_change_lang_cmd.go`,
  `transfer_*_cmd.go` — all wizard/command Action closures with the same dependency profile.
- `rescheduleReminder` (callback_reminder_cmd.go) — needs live `botsfw.WebhookContext` +
  DAL-backed reminder (`dal4reminders.RescheduleReminder`); same integration-harness requirement.
  whyType: external-io; reason: "needs investigation — auto-documented by verifier (engineer did not document)"
- `reportReminderIsActed` (callback_reminer_common.go) — calls `whc.Analytics().Enqueue(...)`;
  requires live `botsfw.WebhookContext`; no useful assertion possible through a mock alone without
  a full integration harness.
  whyType: external-io; reason: "needs investigation — auto-documented by verifier (engineer did not document)"
- `DelayLinkUsersByReceipt`, `delayedLinkUsersByReceipt`, `linkUsersByReceiptNowOrDelay`,
  `linkUsersByReceipt` (callback_receipt_view.go) — depend on DAL
  (`dal4debtus.Default.Receipt.GetReceiptByID`, `facade4debtus.NewReceiptUsersLinker`) and
  the `delaying` task queue; require a full integration harness with faked DAL + queue.
  whyType: external-io; reason: "needs investigation — auto-documented by verifier (engineer did not document)"
