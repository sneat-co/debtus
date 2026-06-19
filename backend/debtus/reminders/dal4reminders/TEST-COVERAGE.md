# TEST-COVERAGE.md — pkg/reminders/dal4reminders

## Coverage: 100%

No production seams were added to this package.

## Test approach

Error paths were covered using:
- `fakeQueryExecutor` — implements `dal.QueryExecutor` and injects errors into `ExecuteQueryToRecordsReader` to test `GetDueReminderIDs` error branches.
- `errorRecordsReader` — implements `dal.RecordsReader` returning an error from `Next()` to test the `SelectAllIDs` error branch.
- `gomock` `MockDB` / `MockReadwriteTransaction` — used to inject errors in `Get` and `Set` calls within transactions for `SetReminderIsSent`, `SetReminderIsSentInTransaction`, and `SetReminderStatus`.
- `nilDataReminder` helper — constructs a `dbo4reminders.Reminder` with `Data == nil` (bypassing `NewReminder`'s nil-to-new conversion) to exercise the re-fetch branch in `SetReminderIsSentInTransaction`.
