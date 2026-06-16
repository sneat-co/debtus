# TEST-COVERAGE.md

## Coverage metrics

| Metric | Value |
|--------|-------|
| Pre-run coverage | 21.8% |
| Post-run coverage | 95.6% |
| Uncovered statements remaining | 22 |

## Seams added

None. All coverage was achieved through direct unit tests only.

## Documented gaps

### defensive/unreachable — panic branches that require impossible inputs

**`bill_common.go:71-78` — `(*BillCommon).AddOrGetMember` existing-member path**
- Lines 72-78 (`else if member = billMembers[index]; member.ID != m.ID { panic }` and `if member.ID == "" { panic }`)
- `billMembers` is always nil in the function (it is never populated from `entity.Members`), so the else-branch always panics with an index-out-of-range, and the `member.ID == ""` check is never reached.
- Refactor required: fix the function to initialize `billMembers` from `entity.GetBillMembers()` before the if/else.

**`bill_owes_calculations.go:55-56` — `updateMemberOwesForEqualSplit` remainder > 1 panic**
- Exhaustive search over all valid Decimal64p2 inputs up to 10000 units and 20 members confirms this condition is mathematically unreachable via integer arithmetic.
- Refactor required: remove the panic or replace with a comment; it is dead code.

**`bill_owes_calculations.go:115-122` — `updateMemberOwesForSplitByShares` remainder loop**
- The `for remainder > 1 || remainder < -1` loop body is never entered; the initial `perShareBy100` calculation always produces a remainder in `[-1, 1]` for any valid Decimal64p2 input.
- Refactor required: same as above — remove or document as dead code.

**`splitus_space_dbo.go:50-51` — `(*SplitusSpaceDbo).AddOrGetMember` index mismatch panic**
- After `append(groupMembers, member)`, `len(groupMembers)-1` always equals `index`. This panic is unreachable.
- Refactor required: remove the unreachable invariant check.

**`splitus_space_dbo.go:54-55` — `(*SplitusSpaceDbo).AddOrGetMember` existing-member ID mismatch panic**
- Requires `groupMembers[index].ID != m.ID` which can only occur if the `Members` slice is internally inconsistent. Not producible by normal code paths.
- Refactor required: remove or guard with a clearer validation error.

**`splitus_space_dbo.go:57-58` — `(*SplitusSpaceDbo).AddOrGetMember` empty member ID panic**
- `AddOrGetMember` always generates a random ID for new members; `member.ID` is never empty after return.
- Refactor required: remove unreachable check.

**`splitus_space_dbo.go:102-103` — `AddOrGetMember` random ID collision loop**
- Requires two randomly generated IDs to collide, which is practically impossible given `MemberIdLen` entropy.
- Refactor required: not needed; the check is a reasonable defensive guard but unreachable in tests.

**`splitus_space_dbo.go:108-109` — `AddOrGetMember` 100-attempt exhaustion panic**
- Requires 100 consecutive random ID collisions. Practically impossible.
- Refactor required: same as above.

### error-path — always-nil error returns

**`bills_holder.go:66-68` — `(*BillsHolder).AddBill` SetOutstandingBills error path**
- `SetOutstandingBills` assigns a field and always returns `nil`. The `if err != nil` branch is unreachable.
- Refactor required: change `SetOutstandingBills` to accept context + storage to produce real errors, or remove the error return.

**`splitus_space_dbo.go:150-152` — `(*SplitusSpaceDbo).RemoveBill` SetOutstandingBills error path**
- Same as above: `SetOutstandingBills` never errors.
- Refactor required: same as `AddBill` error path.

### external-io — json.Marshal/dal.NewKeyWithOptions panic branches

**`bill_history.go:64-65` — `(*BillsHistoryDbo).SetBillSettlements` json.Marshal panic**
- `json.Marshal` on `[]BillSettlementJson` (all basic types) never fails.
- Refactor required: use seam (`var jsonMarshal = json.Marshal`) to inject a failing marshaler in tests.

**`bill_history.go:147-148` — `NewBillsHistory` dal.NewKeyWithOptions panic**
- `dal.NewKeyWithOptions` with `WithRandomStringID` never returns an error.
- Refactor required: inject a seam for the key factory.

**`group.go:38-39` — `NewGroupKey` dal.NewKeyWithOptions panic**
- Same as above.
- Refactor required: same.

**`group.go:77-78` — `(*GroupDbo).SetTelegramGroups` json.Marshal panic**
- `json.Marshal` on `[]GroupTgChatJson` never fails.
- Refactor required: use seam (`var jsonMarshal = json.Marshal`).
