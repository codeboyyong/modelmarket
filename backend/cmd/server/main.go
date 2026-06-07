package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"

	"model-market/backend/internal/config"
	"model-market/backend/internal/httpapi"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel()}))
	logger.Info("app_starting", "MM_APP_ENV", cfg.AppEnv, "dev_mode", cfg.DevMode, "db_url", cfg.DatabaseURL, "db_ssl_mode", cfg.DBSSLMode)

	db, err := sql.Open(cfg.SQLDriverName(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("database_open_failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	var redisHealth httpapi.RedisPinger
	if cfg.RedisEnabled {
		redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
		defer redisClient.Close()
		redisHealth = redisPinger{client: redisClient}
	}

	app := &httpapi.App{
		Config: cfg,
		DB:     db,
		Redis:  redisHealth,
		Logger: logger,
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           requestIDMiddleware(loggingMiddleware(logger, app.Routes())),
		ReadHeaderTimeout: 10 * time.Second,
	}

	logger.Info("server_starting", "addr", cfg.HTTPAddr, "MM_APP_ENV", cfg.AppEnv, "dev_mode", cfg.DevMode, "db_driver", cfg.DBDriver, "db_ssl_mode", cfg.DBSSLMode, "redis_enabled", cfg.RedisEnabled)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server_failed", "error", err)
		os.Exit(1)
	}
}

func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		logger.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", r.Header.Get("X-Request-ID"),
		)
	})
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", time.Now().UnixNano(), r.RemoteAddr)))
			requestID = hex.EncodeToString(sum[:])[:16]
		}
		w.Header().Set("X-Request-ID", requestID)
		r.Header.Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

type redisPinger struct {
	client *redis.Client
}

func (p redisPinger) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
