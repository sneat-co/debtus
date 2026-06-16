# Debtus backend

Go module `github.com/sneat-co/debtus/backend`.

Scaffold only — exposes a `/health` endpoint. Debtus domain endpoints are not
implemented here yet (the live debtus Go domain remains in sneat-go).

## Requirements
- Go 1.26+

## Run
    cd backend
    go run ./cmd/debtusd        # listens on :8080 (override with DEBTUS_ADDR)

## Test
    cd backend
    go test ./...
