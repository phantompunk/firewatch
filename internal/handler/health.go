package handler

import (
	"context"
	"encoding/json"
	"net/http"
)

type pinger interface {
	Ping(ctx context.Context) error
}

// Health returns a health check handler that verifies database connectivity.
func Health(db pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		code := http.StatusOK

		if err := db.Ping(r.Context()); err != nil {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": status})
	}
}
