# TEST-COVERAGE.md — pkg/modules/splitus

## Coverage metrics

| Metric | Value |
|--------|-------|
| Pre-run coverage | 85.7% |
| Post-run coverage | 85.7% |
| Uncovered statements remaining | 1 |

## Seams added

None.

## Documented gaps

### `Module` — module.go:23-25 (HTTP adapter closure body)

**whyType:** external-io

**Why uncoverable:** The inner closure `func(writer http.ResponseWriter, request *http.Request) { handler(request.Context(), writer, request) }` is the HTTP adapter that bridges `extension.HTTPHandleFunc` to `strongoapp.HttpHandlerWithContext`. It is only executed when a live HTTP request hits the registered route. Calling it in a unit test would invoke `handleCreateBill` or `handleGetBill` which require authenticated database access (Firestore).

**Refactor required:** Extract the adapter closure into a named helper function, e.g. `func adaptHandler(handler strongoapp.HttpHandlerWithContext) http.HandlerFunc`. That function can then be tested in isolation with a `httptest.ResponseRecorder`.
