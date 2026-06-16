# TEST-COVERAGE: models4debtus

## Coverage metrics

| Run | Coverage | Uncovered statements |
|-----|----------|----------------------|
| Pre-run | ~97.2% | 18 |
| Post-run | 97.4% | 13 |

## Seams added

- `facade.GetSneatDB` (pre-existing package-level var in sneat-go-core) is used directly in tests to cover the `tx==nil` branch of `GetDebtusSpace` and `TransfersFromQuery`.

## Documented Gaps

All remaining uncovered statements are dead code or impossible error paths under the seam-only production-code rule.

### 1. `debtus_contact.go:164-166` — `WithCounterpartyFields.Validate()` error branch

`WithCounterpartyFields.Validate()` is defined to unconditionally return `nil`.
The `if err != nil { return err }` guard after calling it can never be true.

**Refactor required**: Change `WithCounterpartyFields.Validate()` to actually validate
its fields so that there exist inputs for which it returns a non-nil error.

### 2. `debtus_user_contact_json.go:137-139` — `BalanceWithInterest` error return

`updateBalanceWithInterest` is called with `failOnZeroBalance=false`.
When that flag is false the function never returns an error (it only returns
`ErrBalanceIsZero` when `failOnZeroBalance=true`), so the `if err != nil { return }`
branch is structurally unreachable.

**Refactor required**: Change `updateBalanceWithInterest` to return an error for a
non-`ErrBalanceIsZero` condition when `failOnZeroBalance=false`, or restructure
`BalanceWithInterest` to propagate a different error type.

### 3. `receipt.go:143-146` — `r.Status == ""` check (dead code)

`ValidateString` at line 120 already returns an error when `Status` is not one of the
four known `ReceiptStatuses` values — which includes the empty string. The explicit
`if r.Status == ""` check at line 143 is therefore dead: control never reaches it with
an empty status.

**Refactor required**: Remove the redundant `if r.Status == ""` check at line 143.

### 4. `transfer.go:569-572` — second `CreatorUserID` guard (dead code)

The first guard at line 470 (`if t.CreatorUserID == ""`) already returns early for an
empty creator ID. The later guard (`if t.CreatorUserID <= ""`) at line 569 cannot be
reached with an empty string because the function already returned.

**Refactor required**: Remove the redundant second `CreatorUserID` guard at line 569.

### 5. `transfer.go:617-619` — `onSaveSerializeJson()` error path (unreachable)

Before line 617, `t.From()` and `t.To()` are called at lines 574–575. Both panic if
their respective counterparty is nil with empty JSON. Therefore if `onSaveSerializeJson`
would return an error (e.g. `from == nil && FromJson == ""`), the function already panicked
earlier. The `if err != nil { return }` guard at line 617 is unreachable.

**Refactor required**: Restructure `Validate` to use the error-returning path of
`onSaveSerializeJson` instead of the panic path of `From()/To()`.

### 6. `transfer.go:621-624` and `626-629` — `FromJson`/`ToJson` empty checks (dead code)

After a successful `onSaveSerializeJson()` call, both `FromJson` and `ToJson` are always
non-empty (set inside that function when `from/to != nil`). These post-call guards are dead code.

**Refactor required**: Remove these redundant guards.

### 7. `transfer_fromto.go:36-37` — `TransferCounterpartyInfo.String()` marshal error (dead code)

`json.Marshal` cannot fail for `TransferCounterpartyInfo` (all string/int/SpaceID fields).
The panic branch is defensive dead code.

**Refactor required**: No realistic refactor needed; this is defensive dead code.

### 8. `transfer_fromto.go:197-198` and `206-208` — `onSaveSerializeJson` marshal panics/errors (dead code)

Same reason as gap 7: `json.Marshal` on `*TransferCounterpartyInfo` never fails for normal
Go field types. Both the panic (line 197-198) and the error return (line 206-208) are dead.

**Refactor required**: No realistic refactor needed.

### 9. `transfer_return.go:74-76` — integrity check `len(returns) != ReturnsCount` (dead code)

`GetReturns()` itself panics when `len(t.returns) != t.ReturnsCount` (line 41-42) or when
the JSON-decoded length mismatches (line 51-52). Therefore `GetReturns()` can only return
successfully when `len(returns) == t.ReturnsCount`. The guard at `AddReturn` line 74-76 is
a second layer that can never trigger.

**Refactor required**: Remove the redundant integrity check at AddReturn line 74-76, or
restructure `GetReturns` to return an error instead of panicking.

### 10. `transfer_return.go:104-106` — `json.Marshal(returns)` error (dead code)

`json.Marshal` cannot fail for `[]TransferReturnJson` (all time.Time/decimal/string fields).

**Refactor required**: No realistic refactor needed.

### 11. `with_groups.go:43-44` — `SetActiveGroups` marshal panic (dead code)

`json.Marshal` cannot fail for `[]UserGroupJson` (all primitive string/int fields).

**Refactor required**: No realistic refactor needed.
