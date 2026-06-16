# Coverage metrics
- Pre-run: 0.0% (13 uncovered statements)
- Post-run: 84.6% (2 uncovered statements)

# Seams added
None.

# Documented gaps

## defensive/unreachable
- `NewSplitusSpaceAction` (splitus_common.go:22): `return f(whc, splitusSpace)`
  is unreachable because `GetSplitusSpaceEntryByCallbackUrl` is an unimplemented
  stub that ALWAYS returns an error, so the closure always returns early at the
  `if err != nil` guard. Covering the success path requires implementing
  `GetSplitusSpaceEntryByCallbackUrl` (production logic change, out of scope) or
  introducing an interface seam to override it.
- `NewSplitusSpaceCallbackAction` (splitus_common.go:32): same unreachable
  `return f(...)` for the same reason.
