# Debtus

Debtus — track who owes whom, settle debts, and exchange receipts. A standalone
full-stack product extracted from [sneat-apps](https://github.com/sneat-co/sneat-apps)
and [sneat-go](https://github.com/sneat-co/sneat-go).

This repo hosts two independent toolchains in subdirectories — neither
`package.json` nor `go.mod` lives at the repo root:

- `frontend/` — Nx/Angular/Ionic workspace; hosts `debtus-app` (debtus.app) and
  the `@sneat/ext-debtus-shared` / `@sneat/ext-debtus-internal` libraries.
- `backend/` — Go module (`github.com/sneat-co/debtus/backend`). Scaffold only
  (health endpoint) for now; the live debtus Go domain and Telegram bot remain
  in sneat-go this iteration.

**License:** [AGPL-3.0](LICENSE)
