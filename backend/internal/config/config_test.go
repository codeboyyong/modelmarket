package config

import (
	"log/slog"
	"os"
	"testing"
)

func TestLoadUsesDefaults(t *testing.T) {
	unsetEnv(t, "MM_APP_ENV", "HTTP_ADDR", "MM_DATABASE_URL", "MM_DB_DRIVER", "MM_DB_HOST", "MM_DB_PORT", "MM_DB_NAME", "MM_DB_USER", "MM_DB_PASSWORD", "MM_DB_SSL_MODE", "REDIS_ENABLED", "REDIS_ADDR", "DEV_MODE", "LOG_LEVEL")

	cfg := Load()

	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.RedisAddr != "localhost:6379" {
		t.Fatalf("RedisAddr = %q", cfg.RedisAddr)
	}
	if cfg.RedisEnabled {
		t.Fatal("RedisEnabled default should be false")
	}
	if cfg.AppEnv != "dev" {
		t.Fatalf("AppEnv = %q", cfg.AppEnv)
	}
	if cfg.DBDriver != "postgres" {
		t.Fatalf("DBDriver = %q", cfg.DBDriver)
	}
	if cfg.SQLDriverName() != "pgx" {
		t.Fatalf("SQLDriverName = %q", cfg.SQLDriverName())
	}
	if cfg.DBSSLMode != "disable" {
		t.Fatalf("DBSSLMode = %q", cfg.DBSSLMode)
	}
	if cfg.DatabaseURL != "postgres://model_market:model_market@localhost:5432/model_market?sslmode=disable" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if !cfg.DevMode {
		t.Fatal("DevMode default should be true")
	}
	if cfg.LogLevel() != slog.LevelInfo {
		t.Fatalf("LogLevel = %v", cfg.LogLevel())
	}
}

func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()
	originals := map[string]string{}
	present := map[string]bool{}
	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		originals[key] = value
		present[key] = ok
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, key := range keys {
			if present[key] {
				_ = os.Setenv(key, originals[key])
			} else {
				_ = os.Unsetenv(key)
			}
		}
	})
}

func TestLoadReadsEnvironment(t *testing.T) {
	t.Setenv("MM_APP_ENV", "qa")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("MM_DATABASE_URL", "postgres://example")
	t.Setenv("MM_DB_DRIVER", "postgresql")
	t.Setenv("REDIS_ENABLED", "true")
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("DEV_MODE", "false")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("OBJECT_STORAGE_DIR", "/tmp/assets")

	cfg := Load()

	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.DatabaseURL != "postgres://example" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.SQLDriverName() != "pgx" {
		t.Fatalf("SQLDriverName = %q", cfg.SQLDriverName())
	}
	if !cfg.RedisEnabled {
		t.Fatal("RedisEnabled should read true")
	}
	if cfg.DevMode {
		t.Fatal("DevMode should read false")
	}
	if cfg.ObjectStorageDir != "/tmp/assets" {
		t.Fatalf("ObjectStorageDir = %q", cfg.ObjectStorageDir)
	}
	if cfg.LogLevel() != slog.LevelDebug {
		t.Fatalf("LogLevel = %v", cfg.LogLevel())
	}
}

func TestLoadBuildsProdDatabaseURLWithSSL(t *testing.T) {
	t.Setenv("MM_APP_ENV", "prod")
	t.Setenv("MM_DATABASE_URL", "")
	t.Setenv("MM_DB_HOST", "db.example.com")
	t.Setenv("MM_DB_PORT", "5432")
	t.Setenv("MM_DB_NAME", "market")
	t.Setenv("MM_DB_USER", "app")
	t.Setenv("MM_DB_PASSWORD", "secret")
	t.Setenv("MM_DB_SSL_MODE", "")

	cfg := Load()

	if cfg.DBSSLMode != "require" {
		t.Fatalf("DBSSLMode = %q", cfg.DBSSLMode)
	}
	if cfg.DatabaseURL != "postgres://app:secret@db.example.com:5432/market?sslmode=require" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
}

func TestLoadAllowsExplicitSSLMode(t *testing.T) {
	t.Setenv("MM_APP_ENV", "prod")
	t.Setenv("MM_DATABASE_URL", "")
	t.Setenv("MM_DB_HOST", "db.example.com")
	t.Setenv("MM_DB_PORT", "5432")
	t.Setenv("MM_DB_NAME", "market")
	t.Setenv("MM_DB_USER", "app")
	t.Setenv("MM_DB_PASSWORD", "")
	t.Setenv("MM_DB_SSL_MODE", "verify-full")

	cfg := Load()

	if cfg.DBSSLMode != "verify-full" {
		t.Fatalf("DBSSLMode = %q", cfg.DBSSLMode)
	}
	if cfg.DatabaseURL != "postgres://app@db.example.com:5432/market?sslmode=verify-full" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
}
