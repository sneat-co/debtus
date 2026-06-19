# TEST-COVERAGE.md — pkg/reminders

## Coverage metrics

| Metric | Value |
|--------|-------|
| Pre-run coverage | 48.0% |
| Post-run coverage | 98.6% |
| Uncovered statements remaining | 3 |

(Note: `TestSeamDefaults` now also covers 8 previously-uncovered seam DEFAULT
bodies that wrap external dependencies but return/panic quickly offline
— createSendReminderTask, getDueReminderIDs, getTransferByID,
sendReminderToUserFn, sendReminderByTelegramFn, getBotSettingsByCode,
newTgBotAPIFromSettings, reminderSent, delaySetChatIsForbiddenFn,
setReminderIsSentInTx, delaySetReminderIsSent, tgBotAPISend. The earlier
run's "1 uncovered statement" metric under-counted: the seam-default bodies
were never executed. Only the 2 seam defaults that BLOCK on the network plus
the 1 dead-code branch remain.)

## Seams added to production code

| File | Seam var | Replaces |
|------|----------|---------|
| `cron_handler.go` | `getDueReminderIDs` | `dal4reminders.GetDueReminderIDs` |
| `cron_handler.go` | `createSendReminderTask` | `debtusdal.CreateSendReminderTask` |
| `taskqueu_handler.go` | `getTransferByID` | `facade4debtus.Transfers.GetTransferByID` |
| `taskqueu_handler.go` | `discardReminder` | `delayed4debtus.DiscardReminder` |
| `taskqueu_handler.go` | `sendReminderToUserFn` | `sendReminderToUser` |
| `taskqueu_handler.go` | `getTelegramChatByUserID` | `delayed4debtus.GetTelegramChatByUserID` |
| `taskqueu_handler.go` | `sendReminderByTelegramFn` | `sendReminderByTelegram` |
| `reminder_by_telegram.go` | `getLocale` | `facade4debtusbot.GetLocale` |
| `reminder_by_telegram.go` | `getBotSettingsByCode` | `botsettings.GetBotSettingsByCode` |
| `reminder_by_telegram.go` | `newTgBotAPIFromSettings` | `tgbotapi.NewBotAPIWithClient` |
| `reminder_by_telegram.go` | `tgBotAPISend` | `bot.Send` |
| `reminder_by_telegram.go` | `reminderSent` | `analytics2debtus.ReminderSent` |
| `reminder_by_telegram.go` | `delaySetChatIsForbiddenFn` | `DelaySetChatIsForbidden` |
| `reminder_by_telegram.go` | `setReminderIsSentInTx` | `dal4reminders.SetReminderIsSentInTransaction` |
| `reminder_by_telegram.go` | `delaySetReminderIsSent` | `delay4reminders.DelaySetReminderIsSent` |

## Documented gaps

### `sendReminderByEmail` — line 58-62 (unreachable branch)

**Function:** `sendReminderByEmail` in `reminder_by_email.go:58`

**whyType:** unreachable

**Why uncoverable:** The `if err != nil` block at line 58 checks `err` after lines 52–56, which either return early at line 54 (when `DelaySetReminderIsSent` fails) or set `err = nil` (when `SetReminderIsSent` succeeds). After line 56, `err` is always nil — the block is logically dead code left over from an earlier version.

**Refactor required:** Remove the dead `if err != nil` block (lines 58–62) or preserve the original send error in a separate variable before line 52.

### `discardReminder` and `getTelegramChatByUserID` seam DEFAULT bodies — external-io

**Functions:** the default bodies of `var discardReminder` (taskqueu_handler.go:31)
and `var getTelegramChatByUserID` (taskqueu_handler.go:41).

**whyType:** external-io

**Why uncoverable:** Both seam defaults delegate to `delayed4debtus.*`
operations that make BLOCKING network/RPC calls offline (Cloud Tasks / backend
lookups with no fast-fail path). Calling them in a unit test hangs until the
test timeout. The other 12 seam defaults are covered in `TestSeamDefaults`
because they return or panic immediately offline; these two do not.

**Refactor required:** Push the seam one level deeper (inject the HTTP/Cloud
Tasks client into `delayed4debtus.DiscardReminder` /
`GetTelegramChatByUserID`) so the network call can be stubbed, or wrap them so
they fail fast on a dead/closed context.
