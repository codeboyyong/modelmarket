package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"model-market/backend/internal/config"
	"model-market/backend/internal/httpapi"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel()}))
	db, err := sql.Open(cfg.SQLDriverName(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("worker_database_open_failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)

	app := &httpapi.App{Config: cfg, DB: db, Logger: logger}
	if cfg.ObjectStorageProvider == "s3" {
		store, storageErr := httpapi.NewS3ObjectStore(context.Background(), cfg)
		if storageErr != nil {
			logger.Error("worker_s3_initialization_failed", "error", storageErr)
			os.Exit(1)
		}
		app.ObjectStore = store
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	logger.Info("async_worker_started")
	if err := app.RunAsyncWorker(ctx, 2*time.Second); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("async_worker_stopped", "error", err)
		os.Exit(1)
	}
	logger.Info("async_worker_stopped")
}
