# TEST-COVERAGE.md — facade4splitusbot

## Coverage metrics

| | Coverage | Uncovered statements |
|---|---|---|
| Pre-run | 3.7% | ~52 |
| Post-run | 93.8% | ~5 |

## Seams added

None. All testing uses the existing `facade.GetSneatDB` var seam (already a `var` in the upstream package) and the existing `facade4splitus.DelayerUpdateGroupUsers` / `DelayerUpdateContactWithGroups` / `DelayerUpdateUserWithGroups` delayer vars.

## Documented gaps

### defensive/unreachable

**`CreateGroup` — success path (line 96-97)**
- `logus.Infof(ctx, "GroupEntry created…")` and the trailing `return` are only reached when the transaction succeeds.
- The transaction body immediately returns `errors.New("CreateGroup is not implemented")`, making the success path structurally unreachable without removing or replacing that stub error.
- Refactor required: remove the stub error and implement (or delete) the transaction body.

**`delayedUpdateContactWithGroup` — success return (line 249)**
- The final `return` (no-error path) is only reached when `UpdateContactWithGroups` returns nil.
- `UpdateContactWithGroups` always returns `errors.New("UpdateContactWithGroups not implemented")`.
- Refactor required: implement `UpdateContactWithGroups`.

### external-io / production-bug

**`delayedUpdateUserWithGroups` — `group.Record.Error()` branch (lines 191-193)**
- The production code declares `var splitusSpaceRecords []dal.Record` (nil) and passes it to `tx.GetMulti` instead of the `groups2add[i].Record` slice. As a result, `groups2add[i].Record` is never fetched and `group.Record.Error()` always returns nil (record was never retrieved), making the `return err` branch unreachable.
- Refactor required: pass `[]dal.Record{groups2add[i].Record for i in range}` to `tx.GetMulti`; fix is outside the seam-only rule.
