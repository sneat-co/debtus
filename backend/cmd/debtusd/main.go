// Command debtusd is the debtus backend service.
//
// This is a scaffold: it currently exposes only a health endpoint. Debtus
// domain endpoints are intentionally not implemented yet — the live debtus Go
// domain remains in sneat-go for now.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/sneat-co/debtus/backend/internal/health"
)

func main() {
	addr := os.Getenv("DEBTUS_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	mux := http.NewServeMux()
	mux.Handle("GET /health", health.Handler())

	log.Printf("debtusd listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("debtusd failed: %v", err)
	}
}
