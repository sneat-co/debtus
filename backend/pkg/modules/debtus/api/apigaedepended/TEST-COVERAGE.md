# TEST-COVERAGE.md — apigaedepended

## Coverage metrics

| Run       | Coverage | Uncovered statements |
|-----------|----------|----------------------|
| Pre-run   | 0.0%     | all                  |
| Post-run  | 100.0%   | 0                    |

## Seams added

- `handleFunc` (existing package-level var) — replaced `http.HandleFunc` to capture route registrations in tests.
- `dal4debtus.HttpAppHost` (package-level var) — injected `stubAppHost` so `InitApiGaeDepended` can call `HandleWithContext` without panicking on nil.
