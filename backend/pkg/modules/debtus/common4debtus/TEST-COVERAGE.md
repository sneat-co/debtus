# TEST-COVERAGE.md — common4debtus

## Coverage metrics

| Run       | Coverage | Uncovered statements |
|-----------|----------|----------------------|
| Pre-run   | ~60%     | many                 |
| Post-run  | 99.4%    | 1                    |

## Seams added

None — `token4auth.IssueBotToken` was already a package-level var seam in `sneat-core-modules`.

## Documented gaps

### dead-code

**Function:** `newReceiptTextBuilder` — receipt_text.go:74-75

**Why unreachable:** The `else` branch at line 73 is entered only when `Direction()` returns `TransferDirection3dParty` (i.e. creator is a third party). Inside that branch, line 74 checks `if showReceiptTo != ShowReceiptToCreator && showReceiptTo != ShowReceiptToCounterparty`. However, this condition can never be true: the `switch showReceiptTo` block above (lines 59-66) has already panicked for any value that is not `ShowReceiptToCreator` or `ShowReceiptToCounterparty`. Therefore line 75 (`panic("Unknown ShowReceiptTo: ...")`) is structurally dead — it could only fire if `showReceiptTo` is some other value, but execution would have already panicked on line 65 before reaching line 74.

**Refactor required:** No — this is defensive code that was already made unreachable by an earlier guard. Removing the dead inner `if` would require modifying production logic, which is out of scope for test coverage work.
