# TEST-COVERAGE: dto4debtus

## Coverage metrics

| Run | Coverage | Uncovered statements |
|-----|----------|----------------------|
| Pre-run | 95.5% | 1 |
| Post-run | 100.0% | 0 |

## Seams added

- `var jsonMarshal = json.Marshal` added in `dto.go` — allows tests to override marshalling to force an error, covering the `TransferDto.String()` error branch.

## Documented Gaps

None — 100% statement coverage achieved.
