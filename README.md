# Debtus

Debtus — track who owes whom, settle debts, and exchange receipts — and its
sibling **Splitus** (split shared bills). A standalone full-stack home for both
products, extracted from [sneat-apps](https://github.com/sneat-co/sneat-apps) and
[sneat-go](https://github.com/sneat-co/sneat-go).

This repo hosts two independent toolchains in subdirectories — neither
`package.json` nor `go.mod` lives at the repo root:

- `frontend/` — Nx/Angular/Ionic workspace with two mini-apps, `debtus-app`
  (debtus.app) and `splitus-app` (splitus.app), plus the
  `@sneat/ext-debtus-shared` / `@sneat/ext-debtus-internal` libraries (currently
  shared by both apps).
- `backend/` — single Go module (`github.com/sneat-co/debtus/backend`) housing
  both products as top-level packages: `debtus/`, `splitus/`, and `bots/`
  (`debtusbot` / `splitusbot` / `anybot`). debtus and splitus are bidirectionally
  coupled, so they live in one module rather than two.

CI is the shared [`sneat-co/cicd`](https://github.com/sneat-co/cicd) reusable
workflow (backend + frontend in parallel, then per-app e2e).

**License:** [AGPL-3.0](LICENSE)
