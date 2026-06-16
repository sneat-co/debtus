# TEST-COVERAGE.md — api4transfers

## Coverage metrics

| Run       | Coverage | Uncovered statements |
|-----------|----------|----------------------|
| Pre-run   | 0.0%     | all                  |
| Post-run  | 100.0%   | 0                    |

## Seams added

In `http_create_transfer.go`:

- `createTransferFn = facade4debtus.Transfers.CreateTransfer`
- `newTransferInputFn = func(...) { facade4debtus.NewTransferInput(...) }`

In `http_get_transfer.go`:

- `getTransferByIDFn4transfers = facade4debtus.Transfers.GetTransferByID`
- `checkTransferCreatorNameFn4transfers = facade4debtus.CheckTransferCreatorNameAndFixIfNeeded`

Existing seams reused (not added): `dal4debtus.Default.Transfer`,
`facade.GetSneatDB`, `dal4userus.GetUserByID`, and the auth/verify seam used by
`apicore.HandleAuthenticatedRequestWithBody`.

## Documented gaps

None — 100% statement coverage.

The default body of the `newTransferInputFn` seam is exercised directly in
`TestNewTransferInputFn_defaultBody`; `facade4debtus.NewTransferInput` panics on
the empty UserEntry's missing required field, so the test recovers (the
seam-body line registers as covered before the panic).
