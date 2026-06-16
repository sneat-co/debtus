# Test coverage notes for `facade4splitus`

## Coverage metrics

| Metric | Pre-run | Post-run |
|---|---|---|
| Statement coverage | 45.6% | 98.8% |
| Uncovered statements | 324 | 7 |

All 7 remaining uncovered statements are structurally unreachable
(defensive/unreachable dead code); details below.

## Seams added

All seams are package-level `var` funcs in `seams.go` that wrap previously
hard-wired calls; call sites were swapped to use the vars. Production behavior
is unchanged; tests substitute failure modes that are otherwise unreachable.

| Seam | Wraps | Unlocks |
|---|---|---|
| `applyBillBalanceDifference` | `(*SplitusSpaceDbo).ApplyBillBalanceDifference` | The success path after the balance application in `AssignBillToGroup` (bill_facade.go:85-87) and the post-apply paths in `AddBillMember` (603-617). The wrapped production method is a stub that always returns "not implemented yet". |
| `billAddOrGetMember` | `(*BillDbo).AddOrGetMember` | Everything in `AddBillMember` after line 550. The wrapped production method panics on every input (its `billMembers` named return is never assigned, so both the new-member and existing-member paths blow up). |
| `splitusSpaceAddBill` | `(*SplitusSpaceDbo).AddBill` | The `AddBill` error branches in `RestoreBill` (696-698) and `delayedUpdateGroupWithBill` (group_dal.go:26-28). `AddBill` can only fail via `SetOutstandingBills`, which never returns an error. |
| `splitusSpaceRemoveBill` | `(*SplitusSpaceDbo).RemoveBill` | The `RemoveBill` error branch in `DeleteBill` (653-655); same never-fails reasoning. |
| `setUserOutstandingBills` | `(*SplitusUserDbo).SetOutstandingBills` | The error branch in `delayedUpdateUserWithBill` (bill_delays.go:124-126); the wrapped method never returns an error. |
| `setSpaceGroupMembers` | `(*SplitusSpaceDbo).SetGroupMembers` | The "members not changed" branch in `Settle2members` (settle2members.go:200-202); the wrapped method always returns exactly one update. |
| `newBillKey` | `dal.NewKeyWithOptions(BillKind, WithRandomStringID(...))` | The key-generation error branch in `InsertBillEntity` (741-743); `NewKeyWithOptions` cannot fail for a non-empty collection with the random-string option. |
| `getBillEntryByID` | `GetBillByID` (used by the goroutine in `delayedUpdateUserWithBill`) | The `bill.Data == nil` branch (bill_delays.go:68-70), which is impossible with the real `GetBillByID` (it always allocates `Data` before any fallible call that returns nil error). |
| `isValidBillSplit` | `models4splitus.IsValidBillSplit` | The `SplitMode == ""` branch in `CreateBill` (114-117), which is dead with the real validator because `IsValidBillSplit("")` is false and the earlier check at line 106 already returns. |

No other production changes were made.

## Remaining uncovered statements (7)

### defensive/unreachable

1. **bill_delays.go:32-34 (1 stmt)** — inside `delayedUpdateUsersWithBill`:
   ```go
   if err2 := delayerUpdateUserWithBill.EnqueueWork(...); err != nil {
       err = err2
   }
   ```
   The condition tests the *named return* `err`, not `err2`. `err` is only
   ever assigned inside this dead branch, so it is always nil when evaluated
   and the body can never execute (production bug: the condition should be
   `err2 != nil`). No seam can flip a condition over a local named return.
   Required refactor: fix the condition to test `err2`.

2. **bill_facade.go:209-212 (2 stmts)** — inside `CreateBill`'s member loop:
   ```go
   if member.UserID == "" {
       if len(member.ContactByUser) == 0 {
           err = errors.New("bill member is missing ContactByUser ContactID")
           return
       }
   ```
   The enclosing block (line 204) already returned an error when
   `len(member.ContactByUser) == 0`, so this inner duplicate check can never
   be true. Pure control-flow dead code; no injectable dependency exists.
   Required refactor: delete the duplicate guard.

3. **bill_facade.go:264-269 (4 stmts)** — the `case models4splitus.SplitModeShare`
   arm inside the `ensureMemberAmountDeviateWithin1cent` closure. The closure
   is only invoked from the `case models4splitus.SplitModeEqually` arm of the
   outer switch (line 279), so `billEntity.SplitMode` can never equal
   `SplitModeShare` when the closure runs. Dead code; no seam can make one
   variable hold two values. Required refactor: either call the closure from
   the share-mode arm too, or remove the share case from the closure.

## Other notes

- `AddBillMember` statements after `billAddOrGetMember` (lines 554-620) are
  unreachable in production because `(*BillDbo).AddOrGetMember` always panics
  (see seam table); they are covered through the seam. The statements before
  and including the call are additionally covered via `recover()`-wrapped
  tests against the real (panicking) implementation.
- `delayedUpdateUserWithBill` has a production data race: the bill-reading
  goroutine reads the named return `err` (line 55) without synchronization
  while the main flow writes it (line 60). Tests sequence the goroutine's
  read after the function returns via channels in the fake DB `Get` to stay
  race-detector clean.
- `delayedUpdateUserWithBill` would deadlock against dalgo2memory if the bill
  were read from the same DB (the goroutine's `db.Get` read-lock blocks on the
  transaction write-lock while the worker waits on the goroutine). Tests serve
  the bill from a fake `Get` override on the DB wrapper instead.
