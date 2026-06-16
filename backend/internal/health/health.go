// Package health provides the debtus backend health-check handler.
package health

import (
	"encoding/json"
	"net/http"
)

// Handler responds to health checks with HTTP 200 and a small JSON body.
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}
