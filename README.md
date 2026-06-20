# Debtus

Debtus — track who owes whom, settle debts, and exchange receipts — and its
sibling **Splitus** (split shared bills). A standalone full-stack home for both
products, extracted from [sneat-apps](https://github.com/sneat-co/sneat-apps) and
[sneat-go](https://github.com/sneat-co/sneat-go).

This repo hosts two independent toolchains in subdirectories — neither
`package.json` nor `go.mod` lives at the repo root:

- `frontend/` — Nx/Angular/Ionic workspace with two mini-apps, `debtus-app`
  (debtus.app) and `splitus-app` (splitus.app), each backed by its own
  `extension-<ext>-{contract,shared,internal}` library trio (see
  [Frontend](#frontend) below).
- `backend/` — single Go module (`github.com/sneat-co/debtus/backend`) housing
  both products as top-level packages: `debtus/`, `splitus/`, and `bots/`
  (`debtusbot` / `splitusbot` / `anybot`). debtus and splitus are bidirectionally
  coupled, so they live in one module rather than two.

CI is the shared [`sneat-co/cicd`](https://github.com/sneat-co/cicd) reusable
workflow (backend + frontend in parallel, then per-app e2e).

## Frontend

```bash
cd frontend
pnpm install
npx nx run-many -t build lint test
```

### Library structure (extension library-architecture convention)

Both extensions in this repo — **debtus** and **splitus** — follow the
**extension library-architecture** convention: each extension is split into
three libraries by *runtime weight* and *visibility*, so other repos can depend
on a light **contract** instead of the full bundle, and cross-extension calls go
through dependency-inverted `InjectionToken`s rather than direct implementation
imports. The convention is defined in
[`sneat-co/sneat-libs` → `spec/features/extension-library-architecture`](https://github.com/sneat-co/sneat-libs/tree/main/spec/features/extension-library-architecture/README.md).

For each extension `<ext>` ∈ {`debtus`, `splitus`} (with service token
`<EXT>_SERVICE` and provider `provide<Ext>Internal()`):

| Lib | nx tags | Holds | May depend on |
|-----|---------|-------|---------------|
| `@sneat/extension-<ext>-contract` | `type:contract`, `scope:<ext>` | DTOs/types (e.g. `ICreate*RecordRequest`, `CurrencyCode`) + the `<EXT>_SERVICE` `InjectionToken` (`I<Ext>Service`). Runtime-light — no components/services. | other contracts + foundational `@sneat/*` |
| `@sneat/extension-<ext>-shared` | `type:shared`, `scope:<ext>` | The app-facing UI: routing (`spacePagesRoutes`), pages, components. Obtains services via the `<EXT>_SERVICE` token. | `-contract` + foundational — **never `-internal`** |
| `@sneat/extension-<ext>-internal` | `type:internal`, `scope:<ext>` | The concrete service + `provide<Ext>Internal()`. Private implementation. | `-contract` / `-shared` + foundational |

Concretely:
`@sneat/extension-debtus-{contract,shared,internal}` (`DEBTUS_SERVICE`,
`provideDebtusInternal()`) and
`@sneat/extension-splitus-{contract,shared,internal}` (`SPLITUS_SERVICE`,
`provideSplitusInternal()`).

The boundary matrix is enforced by `@nx/enforce-module-boundaries` in
`frontend/eslint.config.mjs` (a `type:shared → type:internal` import fails lint).
Each `-internal` is consumed only by the composition-root **app**, which wires
`provide<Ext>Internal()` at bootstrap
(`frontend/apps/<ext>-app/src/main.ts`) to bind `<EXT>_SERVICE` to the concrete
service.

**License:** [AGPL-3.0](LICENSE)
