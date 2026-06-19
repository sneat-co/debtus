# TEST-COVERAGE.md — analytics2debtus

## Coverage metrics

| Run       | Coverage | Uncovered statements |
|-----------|----------|----------------------|
| Pre-run   | 9.5%     | ~20                  |
| Post-run  | 95.5%    | 1                    |

## Seams added

- `ga.go`: `var newGaBuffer func(ctx context.Context) gamp.Buffer` — replaces the direct `gamp.NewBufferedClient` call so tests can inject a stub `gamp.Buffer` without making real outbound HTTP requests to Google Analytics.

## Documented gaps

### external-io

**Function:** `newGaBuffer` default closure body (ga.go lines 31–33)

**Reason:** The seam var's default closure body calls `gamp.NewBufferedClient(...)` with a real HTTP client. Tests replace `newGaBuffer` entirely with a stub, so the default body is never executed in tests. The default body is production wiring code only reached at runtime.

**Refactor required:** No refactor needed — this is the standard seam pattern. Covering the default body would require either calling it directly (which would make a real outbound GA HTTP request) or restructuring the seam to expose the inner call as a separate injectable.
