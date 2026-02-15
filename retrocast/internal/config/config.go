package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL      string
	RedisURL         string
	JWTSecret        string
	ServerAddr       string
	LogLevel         slog.Level
	LiveKitURL       string
	LiveKitAPIKey    string
	LiveKitAPISecret string
	MinIOEndpoint    string
	MinIOAccessKey   string
	MinIOSecretKey   string
}

func Load() *Config {
	cfg := &Config{
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		RedisURL:         envOrDefault("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		ServerAddr:       envOrDefault("SERVER_ADDR", ":8080"),
		LogLevel:         parseLogLevel(os.Getenv("LOG_LEVEL")),
		LiveKitURL:       os.Getenv("LIVEKIT_URL"),
		LiveKitAPIKey:    os.Getenv("LIVEKIT_API_KEY"),
		LiveKitAPISecret: os.Getenv("LIVEKIT_API_SECRET"),
		MinIOEndpoint:    os.Getenv("MINIO_ENDPOINT"),
		MinIOAccessKey:   os.Getenv("MINIO_ACCESS_KEY"),
		MinIOSecretKey:   os.Getenv("MINIO_SECRET_KEY"),
	}

	var missing []string
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if cfg.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if len(missing) > 0 {
		panic(fmt.Sprintf("required environment variables not set: %s", strings.Join(missing, ", ")))
	}

	return cfg
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
