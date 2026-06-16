# Coverage metrics

Package: `github.com/sneat-co/sneat-go/pkg/bots/botprofiles/debtusbot/delayed4debtus`

| Metric | Value |
|--------|-------|
| Pre-run coverage | 0% (no tests existed) |
| Post-run coverage | **91.1%** |
| Uncovered statements remaining | ~25 |

## Seams added

| File | Seam |
|------|------|
| `delayed.go` | `var editTgMessageTextFn = editTgMessageText` — allows stubbing the Telegram edit-message call in tests |
| `delayed.go` | `var getTelegramBotApiFn = getTelegramBotApiByBotCode` — allows stubbing the BotAPI constructor in tests |
| `delayed.go` | `var sendToTelegramFn = sendToTelegram` — allows stubbing the Telegram send call in tests; now also wired into `editTgMessageText`'s call site (was a direct call to `sendToTelegram`) |
| `delayed.go` | `var getTelegramChatByUserIDFn = GetTelegramChatByUserID` — allows stubbing the TgChat DB lookup in tests |
| `send_receipt_to_counterparty.go` | `var sendReceiptToTelegramChat = sendReceiptToTelegramChatReal` — allows stubbing the receipt-send call in tests |

Note: `dal4debtus.Default.HttpClient` (a pre-existing package-level func field) and `facade.GetSneatDB`/`botsettings.SetBotSettingsProvider` (pre-existing seams) are overridden in tests; these are not new seams.

## Newly covered this run

- `editTgMessageText` success + send-error paths — `botsettings.SetBotSettingsProvider` + `sendToTelegramFn` stub (call site rewired to the existing unused seam).
- `getTelegramBotApiByBotCode` success path — `botsettings` provider + `dal4debtus.Default.HttpClient` override with a `fakeRoundTripper`.
- `sendToTelegram` success + send-error paths — `dal4debtus.Default.HttpClient` override.
- `DiscardReminder` / `DelayedDiscardReminderForTransfer` wrappers — `facade.GetSneatDB` returns a `mock_dal.MockDB` whose `RunReadwriteTransaction` invokes the callback with a `MockReadwriteTransaction` (no `dalgo2memory` nested-tx deadlock).
- `sendReceiptToTelegramChatReal` getBotApi-error / send-error / success(status-update) — `getTelegramBotApiFn` + `fakeRoundTripper`.
- `DelayedOnReceiptSentSuccess` age-based log-level (between 1h and 24h, between 1m and 1h) and the `Status != Sent` path entry (panics — see gap 2).
- `discardReminder` GetMulti-error-with-returnTransferID (line 174) and GetUser-error (line 206).

## Documented Gaps

### 1. DelayedDiscardReminderForTransfer — ErrDuplicate swallow branch unreachable (60%)

**whyType:** error-path
**Function:** `DelayedDiscardReminderForTransfer` (reminder_delays.go:152–160)
**Uncovered lines:** 154–157 (the `errors.Is(err, dal4reminders.ErrDuplicateAttemptToDiscardReminder)` swallow branch)

**Why untestable:** `dal4reminders.SetReminderStatus` never returns `ErrDuplicateAttemptToDiscardReminder` — the `return ErrDuplicateAttemptToDiscardReminder` line is commented out in `set_reminder_status.go:46`. The success path (line 158) is covered; the swallow branch cannot be reached without re-enabling the commented-out production code.

**Refactor required:** Re-enable the `ErrDuplicateAttemptToDiscardReminder` return in `dal4reminders.SetReminderStatus`.

### 2. on_receipt_sent_success.go:74 — production bug: empty TransferData used instead of loaded transfer

**whyType:** error-path
**Function:** `DelayedOnReceiptSentSuccess` (on_receipt_sent_success.go:36)
**Coverage:** 83.0%
**Uncovered lines:** 74–87 (the block that runs when receipt.Status != ReceiptStatusSent)

**Why untestable:** The function declares `var transferEntity models4debtus.TransferData` (line 61) — a zero-value struct — then calls `transferEntity.Counterparty()` (line 74). `Counterparty()` → `Direction()` panics because `transferEntity` was never loaded from DB. The test (`TestDelayedOnReceiptSentSuccess_statusNotSentPanics`) confirms the panic via `recover`. Covering 74–87 requires fixing the bug: use `transfer.Data.Counterparty()` / `transfer.Data.DtDueOn` instead of the zero-value `transferEntity`.

### 2b. on_receipt_sent_success.go:106–109 — log-level branches unreachable (shadowed var)

**whyType:** error-path
**Function:** `DelayedOnReceiptSentSuccess`
**Uncovered lines:** 106–109 (Infof / Warningf log-level selection)

**Why untestable:** Line 104 reads the OUTER `var receipt models4debtus.ReceiptDbo` (line 56, zero-value), not the inner tx-loaded receipt (line 58 shadows it inside the closure). So `receipt.DtCreated` is always the zero `time.Time` → `Before(now-24h)` is always true → always Debugf (line 105). 106–109 are unreachable.

**Refactor required:** Assign the loaded receipt's `DtCreated` to the outer `receipt` var (or hoist the inner receipt out of the closure).

### 3. sendReceiptToTelegramChatReal — getTranslator/render error sub-branches (84.8%)

**whyType:** error-path
**Function:** `sendReceiptToTelegramChatReal` (send_receipt_to_counterparty.go:153)
**Uncovered lines:** 173–175 (`getTranslator` error — dead, never returns error), 178–180 (`RenderTemplate` error)

**Why untestable:** `getTranslator` always returns a nil error, so its error branch is dead. `RenderTemplate` only fails for a malformed template, but `messageToTranslate` is a fixed valid translation constant — its error branch is not reachable from this call site.

### 4. DelayedCreateAndSendReceiptToCounterpartyByTelegram — receipt empty-ID key panic (76.9%)

**whyType:** external-io
**Function:** `DelayedCreateAndSendReceiptToCounterpartyByTelegram` (send_receipt_to_counterparty.go:228)
**Uncovered lines:** 274–276 (`getTranslator` error — dead, never returns error), 284–286 (`tx.Set` error — not injectable under `dalgo2memory`), 289–300 (receipt ID extraction, ParseInt, delaySend call)

**Why untestable:** `NewReceipt("", ...)` creates a record with an incomplete key. After `tx.Set`, `dalgo2memory` does not auto-assign an ID, so line 287 `receipt.Record.Key().ID.(string)` panics (nil interface, not a string). Lines 255–270, 280–287 are now covered: `TestDelayedCreateAndSendReceiptToCounterpartyByTelegram_chatFound_emptyLocale_userSeeded` seeds the toUser so the `localeCode == ""` fallback (line 270) executes, then proceeds through `getTranslator` and `tx.Set` to the line-287 panic (COVER-BEFORE-PANIC). Covering 289+ needs a backend that auto-assigns IDs or a pre-generated receipt ID; 284–286 needs an injectable `tx.Set` error; 274–276 is dead (`getTranslator` never errors).

### 5. GetTelegramChatByUserID — default/>1 result branch unreachable (88.2%)

**whyType:** external-io
**Function:** `GetTelegramChatByUserID` (delayed.go:23)
**Uncovered lines:** 50–52 (`default` >1-result branch)

**Why untestable:** The query has `Limit(1)` and `dalgo2memory` enforces it, so `len(records)` is never >1. The single-result branch (line 42) is covered (COVER-BEFORE-PANIC — line 44 has a pre-existing value-vs-pointer type-assertion bug). The `default` branch requires removing `Limit(1)` or seaming `dal.ExecuteQueryAndReadAllToRecords`.

### 6. DelayedCreateReminderForTransferUser — UserInfoByUserID panics (97.6%)

**whyType:** error-path
**Function:** `DelayedCreateReminderForTransferUser` (reminder_delays.go:33)
**Uncovered lines:** 55–57 (`transferUserInfo.UserID != userID` branch)

**Why untestable:** `UserInfoByUserID` panics for an unknown user ID instead of returning a sentinel, so the `!=` branch is unreachable. Requires changing `UserInfoByUserID` to return `(info, error)`.

### 7. DelayedSendReceiptToCounterpartyByTelegram — error-log sub-branches (91.3%)

**whyType:** external-io
**Function:** `DelayedSendReceiptToCounterpartyByTelegram` (send_receipt_to_counterparty.go:38)
**Uncovered lines:** 57 (non-NotFound `GetTransferByID` error return), 103–105 (`tx.Set` error after forbidden mark), 110–112 (`delayOnReceiptSentSuccess` error log), 123–125 (`getTranslator` error — dead), 130–132 / 137–139 (`delayOnReceiptSendFail` error logs)

**Why untestable:** Line 57 needs a non-NotFound DB error inside the running `dalgo2memory` tx (no write/read error injection). Lines 103–105 need `tx.Set` to fail (no injection). Lines 110–112 / 130–139 need `delayOnReceiptSentSuccess` / `delayOnReceiptSendFail` to return an error while the surrounding seeded state stays valid — both run via the `delaying.VoidWithLog` delayer whose synchronous callback succeeds with seeded records. Line 123 is dead (`getTranslator` never errors).

### 8. DelayedOnReceiptSentSuccess — tx.GetMulti and tx.SetMulti error returns (on_receipt_sent_success.go:65, 79)

**whyType:** external-io
**Function:** `DelayedOnReceiptSentSuccess` (on_receipt_sent_success.go:36)
**Uncovered lines:** 65 (`return err` from `tx.GetMulti` failure), 79 (`return fmt.Errorf(...)` from `tx.SetMulti` failure)

**Why untestable:** `dalgo2memory` does not support injecting read or write errors inside a running transaction. Line 65 needs GetMulti to return an error with seeded records present; line 79 needs SetMulti to fail after a successful GetMulti. Neither is injectable without a mock transaction, and introducing a mockTx here would conflict with the existing `facade.RunReadwriteTransaction` call site.
**Reason:** needs investigation — auto-documented by verifier (engineer did not document)

### 9. sendReceiptToTelegramChatReal — updateReceiptStatus error swallow (send_receipt_to_counterparty.go:219-221)

**whyType:** external-io
**Function:** `sendReceiptToTelegramChatReal` (send_receipt_to_counterparty.go:153)
**Uncovered lines:** 219–221 (`logus.Errorf` + `err = nil` inside `updateReceiptStatus` error path)

**Why untestable:** `updateReceiptStatus` opens an inner `facade.RunReadwriteTransaction`; making it fail requires the in-memory DB's `RunReadwriteTransaction` to return an error, which is not injectable when `facade.GetSneatDB` returns a `dalgo2memory` instance. The success path (receipt status update) is covered by the existing `success_updates_status` test.
**Reason:** needs investigation — auto-documented by verifier (engineer did not document)

### 10. discardReminder — else-if botsettings branch (reminder_delays.go:211-213)

**whyType:** external-io
**Function:** `discardReminder` (reminder_delays.go:162)
**Uncovered lines:** 211–213 (`else if s, sErr := botsettings.GetBotSettingsByCode(...); sErr == nil { reminder.Data.Locale = s.Locale.Code5 }`)

**Why untestable:** This branch is reached when `reminder.Data.Locale` is empty AND `user.Data.PreferredLocale` is also empty. The existing `TestDiscardReminder_TelegramGetUserError` covers the GetUser error return (line 207). Reaching lines 211–213 requires seeding a user with an empty `PreferredLocale` and registering a bot settings provider that returns a bot with a non-empty locale. This requires `botsettings.SetBotSettingsProvider` to be called alongside a seeded user record — not attempted by the existing tests.
**Reason:** needs investigation — auto-documented by verifier (engineer did not document)
