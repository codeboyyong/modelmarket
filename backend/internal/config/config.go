package config

import (
	"log/slog"
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr         string
	DatabaseURL      string
	RedisAddr        string
	DevMode          bool
	MockDataDir      string
	MigrationsDir    string
	ObjectStorageDir string
	LogLevelName     string
	PublicURL        string
}

func Load() Config {
	return Config{
		HTTPAddr:         env("HTTP_ADDR", ":8080"),
		DatabaseURL:      env("DATABASE_URL", "postgres://model_market:model_market@localhost:5432/model_market?sslmode=disable"),
		RedisAddr:        env("REDIS_ADDR", "localhost:6379"),
		DevMode:          envBool("DEV_MODE", true),
		MockDataDir:      env("MOCK_DATA_DIR", "../mock-data"),
		MigrationsDir:    env("MIGRATIONS_DIR", "backend/migrations"),
		ObjectStorageDir: env("OBJECT_STORAGE_DIR", "tmp/storage"),
		LogLevelName:     env("LOG_LEVEL", "info"),
		PublicURL:        env("PUBLIC_URL", "http://localhost:3000"),
	}
}

func (c Config) LogLevel() slog.Level {
	switch c.LogLevelName {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}
