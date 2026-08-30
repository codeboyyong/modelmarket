package main

import (
	"context"
	"database/sql"
	"errors"
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
	if cfg.ObjectStorageProvider == "s3" {
		store, storageErr := httpapi.NewS3ObjectStore(context.Background(), cfg)
		if storageErr != nil {
			logger.Error("s3_initialization_failed", "error", storageErr)
			os.Exit(1)
		}
		app.ObjectStore = store
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           app.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	logger.Info("server_starting", "addr", cfg.HTTPAddr, "MM_APP_ENV", cfg.AppEnv, "dev_mode", cfg.DevMode, "db_driver", cfg.DBDriver, "db_ssl_mode", cfg.DBSSLMode, "redis_enabled", cfg.RedisEnabled)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server_failed", "error", err)
		os.Exit(1)
	}
}

type redisPinger struct {
	client *redis.Client
}

func (p redisPinger) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}
