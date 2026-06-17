package httpapi

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"

	"model-market/backend/internal/config"
)

type RedisPinger interface {
	Ping(context.Context) error
}

type App struct {
	Config config.Config
	DB     *sql.DB
	Redis  RedisPinger
	Logger *slog.Logger
	Client *http.Client
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /readyz", a.ready)
	mux.HandleFunc("GET /api/v1/dev/summary", a.devSummary)
	mux.HandleFunc("POST /api/v1/auth/dev-login", a.devLogin)
	mux.HandleFunc("POST /api/v1/auth/login", a.passwordLogin)
	mux.HandleFunc("POST /api/v1/auth/signup", a.signup)
	mux.HandleFunc("POST /api/v1/auth/change-password", a.changePassword)
	mux.HandleFunc("POST /api/v1/auth/social/dev", a.devSocialLogin)
	mux.HandleFunc("GET /api/v1/models", a.models)
	mux.HandleFunc("GET /api/v1/pricing", a.pricing)
	mux.HandleFunc("GET /api/v1/company-usage", a.companyUsage)
	mux.HandleFunc("GET /api/v1/user-credit-usage", a.userCreditUsage)
	mux.HandleFunc("GET /api/v1/admin/overview", a.adminOverview)
	mux.HandleFunc("POST /api/v1/credits/purchase", a.purchaseCredits)
	mux.HandleFunc("POST /api/v1/payments/stripe/webhook", a.stripeWebhook)
	mux.HandleFunc("GET /api/v1/projects", a.projects)
	mux.HandleFunc("POST /api/v1/projects", a.createProject)
	mux.HandleFunc("GET /api/v1/conversations", a.conversations)
	mux.HandleFunc("POST /api/v1/conversations", a.createConversation)
	mux.HandleFunc("GET /api/v1/conversation-branches", a.conversationBranches)
	mux.HandleFunc("POST /api/v1/conversation-branches", a.createConversationBranch)
	mux.HandleFunc("GET /api/v1/messages", a.messages)
	mux.HandleFunc("GET /api/v1/assets", a.assets)
	mux.HandleFunc("POST /api/v1/assets/upload-intent", a.createUploadIntent)
	mux.HandleFunc("GET /api/v1/mock-s3/", a.mockS3Object)
	mux.HandleFunc("PUT /api/v1/mock-s3/", a.mockS3Object)
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
