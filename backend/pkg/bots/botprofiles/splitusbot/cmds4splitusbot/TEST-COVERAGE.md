# Coverage metrics

| Metric | Value |
|--------|-------|
| Pre-run coverage (committed tests only) | 5.1% |
| Post-run coverage | 90.7% (`go test -cover`; `go tool cover -func` reports 92.4% statement-weighted) |
| Uncovered statements remaining | 97 (78 coverage blocks; one block is the 0-statement empty `default:` case at `bill_join_cmd.go:202`) |

All remaining uncovered statements are documented below, grouped by why-type.
None of them are reachable from unit tests without production changes that go
beyond the seam-only policy (package-level `var name = expr`).

## Seams added

- `bill_common_botcmd.go`: `var getUserGroupID = bothelper.GetUserGroupID` —
  lets tests inject an error from `bothelper.GetUserGroupID` to cover the
  in-group error branch of `billCallbackAction` (the real function can never
  return a non-nil error, see "defensive/unreachable" below).

No other production code was changed. Several pre-existing package-level vars
were used as natural seams from tests: `facade.GetSneatDB` (sneat-go-core),
`delayUpdateBillCards` / `delayUpdateBillTgChatCard` (this package's
`delaying.Delayer` vars), `botsettings.SetBotSettingsProvider`, and
`dal4debtus.Default.HttpClient`.

## Documented gaps

### defensive/unreachable (dead error paths of never-failing dependencies)

`writeBillCardTitle` can only fail if `bytes.Buffer.WriteString` fails, which
never happens; therefore `getBillCardMessageText` never returns an error and
every error branch downstream of it is dead code. Refactor required: make the
title/template rendering injectable (e.g. a package-level
`var renderBillCardTitle`) or stop returning an impossible error.

- `bill_card_2_cmd.go:36` (billCardCommand closure), `:69` (billMembersCommand
  closure), `:209` (ShowBillCard), `:234` (WriteString error), `:247`
  (getBillCardMessageText error propagation)
- `bill_change_payer_cmd.go:25`, `bill_edit_cmd.go:21`,
  `bill_finalize_cmd.go:20`, `bill_delete_cmd.go:53` (restore closure),
  `split_mode_list_cmd.go:21`, `bill_set_currency_cmd.go:52`
- `bill_join_delays.go:63` and `:89` (updateInlineBillCardMessage /
  getBillCardMessageText error branches)
- `inline_choosen.go:153-160` (empty-card-text guard: the footer always
  contains a non-empty horizontal rule, so `m.Text` can never be blank)

Other dead branches:

- `bill_delete_cmd.go:47-50` — the restore closure maps
  `ErrSettledBillsCanNotBeDeleted`, but `RestoreBill` can only return
  `ErrOnlyDeletedBillsCanBeRestored` (copy-paste bug); branch unreachable.
- `common_split.go:119` — `if i = len(members)-1; i < 0` re-check is
  unreachable because `len(members) == 0` already returned earlier.
- `common_split.go:60` — `writeTitle` error return; the only writeTitle
  callback (billShares) delegates to the never-failing `writeBillCardTitle`.
- `currencies.go:27` — `flag == ""` branch of `currencyButton`; every call
  site passes a non-empty flag literal.
- `group_members_cmd.go:69` — "count > 0 but no briefs" mismatch; both values
  are derived from the same contact briefs, so they cannot diverge.
- `group_members_cmd.go:95` — `groupMembersCard` always returns a nil error.
- `inline_choosen.go:207` — `billID == ""` after a successful `(\d+)` regex
  match cannot happen.
- `inline_choosen.go:232` — `bothelper.GetUserGroupID` never returns a
  non-nil error (all paths return `nil` error), so this error branch is dead.
  (The same fact is why the in-package `getUserGroupID` seam was needed to
  cover `billCallbackAction`.)
- `inline_choosen.go:126` — the `defer recover -> re-panic` body inside
  `createBillFromInlineChosenResult`; no statement inside the transaction can
  panic for inputs constructible in a unit test.
- `chat_new_members_cmd.go:66` — `tx.Set` error for the bot-user record; the
  in-memory adapter cannot fail here.
- `bill_join_cmd.go:202` — empty `default:` case (0 statements).

### error-path (would need upstream seams outside this package)

- `bill_set_currency_cmd.go:29`, `split_mode_change_cmd.go:39`,
  `bill_split_cmd.go:55` — `facade4splitus.SaveBill` error branches.
  SaveBill = `tx.Set` + delayer enqueue; neither fails under the in-memory DB
  and the facade4splitus delayer vars are unexported, so an error can only be
  injected with a seam in facade4splitus (outside this package's scope).

### production-bug-unreachable (blocked by bugs the seam-only policy forbids fixing)

- `bill_join_cmd.go:207-223` — everything after
  `facade4splitus.AddBillMember`: `BillCommon.AddOrGetMember` checks
  `index != len(billMembers)-1` against a nil named return value and panics
  whenever a NEW member is added to a bill, so AddBillMember can never return
  for a joining user. Covered up to the panic via recover-wrapped tests.
- `bill_new_cmd.go:97-103` — the create-bill transaction: the bill member is
  constructed with only `Paid` set (no `Name`), so `SetBillMembers` always
  fails with "no name for the members[0]" before the transaction runs.
- `bill_set_currency_cmd.go:42-49` — statements at/after
  `ApplyBillBalanceDifference`, which always returns
  "ApplyBillBalanceDifference is not implemented yet".
- `chat_new_members_cmd.go:86-118` — statements at/after
  `AddUsersToTheGroupAndOutstandingBills`: the hard-coded placeholder space ID
  ("TODO-implement-determining-space-id", 35 chars) makes
  `NewSplitusSpaceEntry` panic (max key length 30). Covered up to the panic
  via a recover-wrapped test.
- `bill_join_cmd.go:176-184` — the in-group default-currency-from-space
  branch: `bothelper.GetSpaceEntryByUrl(whc, nil)` can never succeed with a
  nil URL (the empty-space-ID path returns "not implemented"), so
  `space.Data != nil` is unreachable; the branch body itself is
  `errors.New("not implemented yet")`.
- `currencies.go:118-137` — groupSettingsSetCurrencyCommand callback closure:
  `bothelper.NewSpaceCallbackAction` drops the callback URL (passes nil to
  `GetSpaceEntryByUrl`), so the space can never resolve and the closure is
  never invoked. Refactor required in bothelper (outside scope): pass
  `callbackUrl` through.
- `start_settle_group_cmd.go:26` and `:29` —
  settleGroupAskForCounterpartyCommand Action/CallbackAction closures; same
  `NewSpaceAction`/`NewSpaceCallbackAction` nil-URL problem as above.
- `settings.go:97` — the `groupAction` inner closure in `settingsAction`;
  same `NewSpaceAction` nil-URL problem.
- `group_balance_cmd.go:29` — groupBalanceCommand callback closure:
  `shared_splitus.GetSplitusSpaceEntryByCallbackUrl` always returns
  "not implemented yet", so `NewSplitusSpaceCallbackAction` never invokes it.
- `group_split_cmd.go:40-63` — the spaceSplit `addShares` closure body: the
  members slice passed to `editSplitCallbackAction` is always an empty
  literal, so `getSplitParamsAndCurrentMember` errors before `addShares` can
  run.

### external-io / framework-coupled (require full bot webhook flow)

- `1_register_commands.go:28-38` — the `GetWelcomeMessageText` BotParams
  closure: only invoked from `sharedStartCommandCallbackAction` after
  `runBotSpecificStartCommand`, which calls `StartInBotAction` with empty
  start params and therefore always fails with `ErrUnknownStartParam` before
  the welcome text is requested.
- `1_register_commands.go:40-43` — the `SetMainMenu` BotParams closure: only
  invoked by `setPreferredLocaleAction` in `setPreferredLocaleModeSettings`
  mode, which is never wired to this closure by the current cmds4anybot
  command graph (the start command uses `setPreferredLocaleModeStart`, which
  returns before calling `mainMenuAction`).

## Notes on test infrastructure

- `withMemDB` wraps `dalgo2memory` in a `reentrantDB` so nested
  `facade.RunReadwriteTransaction` calls (e.g. `DeleteBill` inside
  `billCallbackAction`) reuse the outer transaction instead of deadlocking on
  the adapter's non-reentrant mutex.
- The dalgo2memory adapter keys nested records by their last key component, so
  `spaces/<id>/ext/<module>` records collide across spaces; tests that need a
  not-found module record run before any record of that module is seeded.
