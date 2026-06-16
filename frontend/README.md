# debtus frontend

Nx workspace for the debtus frontend: the standalone `debtus-app` and the
publishable `@sneat/ext-debtus-shared` and `@sneat/ext-debtus-internal` libraries.

- **Nx** 22 · **Angular** 21 · **Ionic** 8 · **pnpm**

## Setup

```bash
pnpm install
```

## Common tasks

```bash
pnpm exec nx serve debtus-app          # run the app locally
pnpm exec nx build ext-debtus-shared   # build a publishable library
pnpm exec nx run-many -t lint test build
pnpm exec nx e2e debtus-app-e2e        # end-to-end tests
```

## Layout

```
frontend/
├── apps/
│   └── debtus-app/             # standalone debtus.app (Ionic shell)
└── libs/
    ├── ext-debtus-shared/      # @sneat/ext-debtus-shared (publishable)
    └── ext-debtus-internal/    # @sneat/ext-debtus-internal (publishable)
```

> Projects are generated incrementally during the extraction; see the repo
> root README for the overall plan.
