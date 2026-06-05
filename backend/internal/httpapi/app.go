package httpapi

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/redis/go-redis/v9"

	"model-market/backend/internal/config"
)

type App struct {
	Config config.Config
	DB     *sql.DB
	Redis  *redis.Client
	Logger *slog.Logger
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /readyz", a.ready)
	mux.HandleFunc("GET /api/v1/dev/summary", a.devSummary)
	mux.HandleFunc("POST /api/v1/auth/dev-login", a.devLogin)
	mux.HandleFunc("GET /api/v1/models", a.models)
	mux.HandleFunc("GET /api/v1/projects", a.projects)
	mux.HandleFunc("GET /api/v1/api-keys", a.apiKeys)
	mux.HandleFunc("POST /api/v1/api-keys", a.createAPIKey)
	mux.HandleFunc("DELETE /api/v1/api-keys/", a.revokeAPIKey)
	mux.HandleFunc("POST /api/v1/chat/completions", a.chatCompletions)
	return cors(mux)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
