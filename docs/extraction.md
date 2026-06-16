# Origin: extracted from sneat-apps / sneat-go

This repository was created by extracting the **debtus** extension out of the
sneat monorepos. The authoritative specification and implementation plan live
in the **sneat-apps** repo:

- **Idea spec:** `spec/ideas/extract-debtus-standalone-repo.md`
  (in [sneat-co/sneat-apps](https://github.com/sneat-co/sneat-apps))
- **Implementation plan:** `docs/superpowers/plans/2026-06-16-extract-debtus-standalone-repo.md`
  (in [sneat-co/sneat-apps](https://github.com/sneat-co/sneat-apps))

## What landed in this iteration

- **Frontend** (`frontend/`): the `@sneat/ext-debtus-shared` and
  `@sneat/ext-debtus-internal` libraries plus the standalone `debtus-app`
  (debtus.app) shell. Published to npm at `0.0.1` (built against `@sneat/* ^0.5.0`).
- **Backend** (`backend/`): a scaffold-only Go module
  (`github.com/sneat-co/debtus/backend`) exposing a health endpoint.

sneat-apps now consumes the published `@sneat/ext-debtus-*` packages instead of
local libraries (sneat-apps PR #3409).

## Deliberately deferred

- The real debtus **Go domain** and the **`debtusbot`** Telegram bot remain in
  **sneat-go** for now (see `backend/docs/` for backend-extraction planning).
- `debtus.app` hosting wiring.
