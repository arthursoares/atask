package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// NewRouter and the legacy *http.ServeMux stack were removed in Task 11. Routing
// now lives in RegisterRoutes (routes.go), which registers each handler directly
// on PocketBase's router with per-route auth. The per-handler RegisterRoutes(mux)
// methods are retained solely for the mux-based test fixtures in internal/api.

type healthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

// handleHealth serves GET /health with a 200 and a small JSON body. It is a
// public endpoint (no auth middleware).
func handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{
		Status: "ok",
		Time:   time.Now().UTC().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode health response", "err", err)
	}
}
