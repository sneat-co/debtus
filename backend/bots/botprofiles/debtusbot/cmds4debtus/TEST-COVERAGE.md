# Test Coverage

## Coverage metrics

| Metric | Value |
|--------|-------|
| Pre-run coverage | 11.3% |
| Post-run coverage | 100.0% |
| Uncovered statements remaining | 0 |

## Seams added

| File | Seam | Purpose |
|------|------|---------|
| inline_choosen_result.go | `var urlParse = url.Parse` | Inject URL parse error in tests |
| inline_choosen_result.go | `var onInlineChosenCreateReceipt = dtb_transfer.OnInlineChosenCreateReceipt` | Stub receipt handler |
| inline_query_handler.go | `var inlineSendReceipt = dtb_transfer.InlineSendReceipt` | Stub inline receipt send |
| inline_query_handler.go | `var inlineNewRecord = dtb_inline.InlineNewRecord` | Stub inline new record |

## Documented gaps

None — all statements are covered.
