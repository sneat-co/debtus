# Debtus backend

Single Go module `github.com/sneat-co/debtus/backend` housing both the **debtus**
and **splitus** products (they are bidirectionally coupled, so one module, not two).

Top-level packages:

- `debtus/` — the debtus domain (`github.com/sneat-co/debtus/backend/debtus/...`).
- `splitus/` — the splitus domain (`.../backend/splitus/...`).
- `bots/` — Telegram bots: `debtusbot`, `splitusbot`, `anybot` (`.../backend/bots/...`).
- `cmd/debtusd` — service entrypoint (`/health`).
- `internal/` — internal helpers.

## Requirements
- Go 1.26+

## Run
    cd backend
    go run ./cmd/debtusd        # listens on :8080 (override with DEBTUS_ADDR)

## Test
    cd backend
    go test ./...
