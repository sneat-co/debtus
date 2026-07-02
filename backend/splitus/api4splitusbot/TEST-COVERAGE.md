# TEST-COVERAGE.md — api4splitusbot

## Coverage metrics

| Metric | Value |
|--------|-------|
| Package coverage | 95.7% |

## Seams added

- `var getBillByID = facade4splitus.GetBillByID` in `api_bills.go` — allows tests to stub the GetBillByID call
- `var getContactsByIDs = dal4contactus.GetContactsByIDs` in `api_bills.go`
- `var getUsersByIDs = dal4userus.GetUsersByIDs` in `api_bills.go`
- `var createBill = facade4splitus.CreateBill` in `api_bills.go`
- `var runReadwriteTransaction = facade.RunReadwriteTransaction` in `api_bills.go`
- `var createDebtusTransfer = facade4debtus.Transfers.CreateTransfer` in `api_create_split.go` — lets tests assert per-participant Debtus transfers without real I/O
- `var getBillTransfers = func(ctx, spaceID, billID)` in `api_get_splits.go` — loads the Debtus transfers whose `TransferData.BillIDs` contains the bill ID (the reverse of `CreateTransferRequest.BillID` written by `handleCreateSplit`); settled state is DERIVED from these transfers, never stored on the Bill. The production body is also exercised against the in-memory DB (`TestGetBillTransfers_QueriesByBillID`)
- `var listBillsBySpace = facade4splitus.ListBillsBySpace` in `api_get_splits.go` — list read for `GET /api4splitus/splits`

## Documented gaps

- `handleCreateSplit`: two defensive branches are not exercised — the
  `SetBillMembers` error return (unreachable with named members and a valid
  equal split) and the `CreateTransferInput.Validate()` error return
  (unreachable for inputs the handler itself constructs).
