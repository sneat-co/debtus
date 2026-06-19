# Test Coverage

## Coverage metrics

| Metric | Value |
|--------|-------|
| Pre-run coverage | ~60% |
| Post-run coverage | 98.8% |
| Uncovered statements remaining | 2 |

## Seams added

None.

## Documented gaps

### Structurally dead code after `echoSelection` (ask_invite_contact.go:96, 101)

- **whyType**: defensive/unreachable
- **function**: `askInviteAddressCallbackCommand.CallbackAction` — the lines `return AskInviteAddressEmailCommand.Action(whc)` (line 96) and `return AskInviteAddressSmsCommand.Action(whc)` (line 101) are unreachable.
- **reason**: `echoSelection` (defined inline at line 83) always returns a non-nil error via `return fmt.Errorf("failed to edit callback message: %w", err)` — even when `err` is nil, `fmt.Errorf` wraps it into a non-nil error. Therefore the `if err = echoSelection(...); err != nil { return }` guard always triggers, making the subsequent lines dead.
- **refactorRequired**: The function itself would need to be restructured so `echoSelection` can succeed (return nil) in some cases, or the guard condition changed to `err != nil && !errors.Is(err, ...)`. This is a production logic change, not a seam.
