# TEST-COVERAGE.md — api4splitusbot

## Coverage metrics

| Metric | Value |
|--------|-------|
| Pre-run coverage | 0.0% |
| Post-run coverage | 100.0% |
| Uncovered statements remaining | 0 |

## Seams added

- `var getBillByID = facade4splitus.GetBillByID` in `api_bills.go` — allows tests to stub the GetBillByID call
- `var getDebtusSpaceContactsByIDs = facade4debtus.GetDebtusSpaceContactsByIDs` in `api_bills.go`
- `var getContactsByIDs = dal4contactus.GetContactsByIDs` in `api_bills.go`
- `var getUsersByIDs = dal4userus.GetUsersByIDs` in `api_bills.go`
- `var createBill = facade4splitus.CreateBill` in `api_bills.go`
- `var runReadwriteTransaction = facade.RunReadwriteTransaction` in `api_bills.go`

## Documented gaps

None — 100% statement coverage achieved.
