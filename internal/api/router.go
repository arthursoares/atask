package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/atask/atask/internal/service"
)

// NewRouter constructs and returns the fully wired HTTP handler.
// Public routes (/health and /auth/*) bypass the Auth middleware.
func NewRouter(
	areaHandler *AreaHandler,
	taskHandler *TaskHandler,
	projectHandler *ProjectHandler,
	sectionHandler *SectionHandler,
	tagHandler *TagHandler,
	locationHandler *LocationHandler,
	checklistHandler *ChecklistHandler,
	activityHandler *ActivityHandler,
	viewHandler *ViewHandler,
	eventsHandler *EventsHandler,
	syncHandler *SyncHandler,
	authHandler *AuthHandler,
	authService *service.AuthService,
) http.Handler {
	mux := http.NewServeMux()

	// Health endpoint (no auth required)
	mux.HandleFunc("GET /health", handleHealth)

	// Auth routes (register/login are public; protected auth routes skip via path check in middleware)
	authHandler.RegisterRoutes(mux)

	// All other routes — auth is enforced by the Auth middleware
	areaHandler.RegisterRoutes(mux)
	taskHandler.RegisterRoutes(mux)
	projectHandler.RegisterRoutes(mux)
	sectionHandler.RegisterRoutes(mux)
	tagHandler.RegisterRoutes(mux)
	locationHandler.RegisterRoutes(mux)
	checklistHandler.RegisterRoutes(mux)
	activityHandler.RegisterRoutes(mux)
	viewHandler.RegisterRoutes(mux)
	eventsHandler.RegisterRoutes(mux)
	syncHandler.RegisterRoutes(mux)

	// Middleware stack: RequestID → Logging → Auth → mux
	// Auth skips /health and /auth/* paths automatically.
	var handler http.Handler = mux
	handler = authMiddleware(authService)(handler)
	handler = Logging(handler)
	handler = RequestID(handler)

	return handler
}

// authMiddleware wraps the Auth middleware but skips paths that don't require authentication:
// /health and anything under /auth/.
func authMiddleware(authService *service.AuthService) func(http.Handler) http.Handler {
	inner := Auth(authService)
	return func(next http.Handler) http.Handler {
		protected := inner(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" || strings.HasPrefix(r.URL.Path, "/auth/") {
				next.ServeHTTP(w, r)
				return
			}
			protected.ServeHTTP(w, r)
		})
	}
}

type healthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

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
