package config

import (
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr         string
	AppEnv           string
	DBDriver         string
	DatabaseURL      string
	DBHost           string
	DBPort           string
	DBName           string
	DBUser           string
	DBPassword       string
	DBSSLMode        string
	RedisEnabled     bool
	RedisAddr        string
	DevMode          bool
	ObjectStorageDir string
	AssetBucket      string
	AssetPublicURL   string
	LogLevelName     string
	PublicURL        string
}

func Load() Config {
	appEnv := env("MM_APP_ENV", "dev")
	dbSSLMode := env("MM_DB_SSL_MODE", defaultDBSSLMode(appEnv))
	dbHost := env("MM_DB_HOST", "localhost")
	dbPort := env("MM_DB_PORT", "5432")
	dbName := env("MM_DB_NAME", "model_market")
	dbUser := env("MM_DB_USER", "model_market")
	dbPassword := envAllowEmpty("MM_DB_PASSWORD", "model_market")
	databaseURL := env("MM_DATABASE_URL", "")
	if databaseURL == "" {
		databaseURL = buildPostgresURL(dbHost, dbPort, dbName, dbUser, dbPassword, dbSSLMode)
	}

	return Config{
		HTTPAddr:         env("HTTP_ADDR", ":8080"),
		AppEnv:           appEnv,
		DBDriver:         env("MM_DB_DRIVER", "postgres"),
		DatabaseURL:      databaseURL,
		DBHost:           dbHost,
		DBPort:           dbPort,
		DBName:           dbName,
		DBUser:           dbUser,
		DBPassword:       dbPassword,
		DBSSLMode:        dbSSLMode,
		RedisEnabled:     envBool("REDIS_ENABLED", false),
		RedisAddr:        env("REDIS_ADDR", "localhost:6379"),
		DevMode:          envBool("DEV_MODE", true),
		ObjectStorageDir: env("OBJECT_STORAGE_DIR", "tmp/storage"),
		AssetBucket:      env("MM_ASSET_BUCKET", "model-market-dev-assets"),
		AssetPublicURL:   strings.TrimRight(env("MM_ASSET_PUBLIC_URL", ""), "/"),
		LogLevelName:     env("LOG_LEVEL", "info"),
		PublicURL:        env("PUBLIC_URL", "http://localhost:3000"),
	}
}

func (c Config) SQLDriverName() string {
	switch strings.ToLower(c.DBDriver) {
	case "postgres", "postgresql", "pgx":
		return "pgx"
	default:
		return c.DBDriver
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

func defaultDBSSLMode(appEnv string) string {
	switch strings.ToLower(appEnv) {
	case "prod", "production", "qa", "staging":
		return "require"
	default:
		return "disable"
	}
}

func buildPostgresURL(host, port, dbName, user, password, sslMode string) string {
	userInfo := url.User(user)
	if password != "" {
		userInfo = url.UserPassword(user, password)
	}
	u := url.URL{
		Scheme: "postgres",
		User:   userInfo,
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + dbName,
	}
	query := u.Query()
	query.Set("sslmode", sslMode)
	u.RawQuery = query.Encode()
	return u.String()
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envAllowEmpty(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
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
