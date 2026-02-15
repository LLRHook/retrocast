package config

import "os"

type Config struct {
	DatabaseURL      string
	RedisURL         string
	JWTSecret        string
	ServerAddr       string
	LiveKitURL       string
	LiveKitAPIKey    string
	LiveKitAPISecret string
	MinIOEndpoint    string
	MinIOAccessKey   string
	MinIOSecretKey   string
}

func Load() *Config {
	return &Config{
		DatabaseURL:      envOrDefault("DATABASE_URL", "postgres://retrocast:password@localhost:5432/retrocast?sslmode=disable"),
		RedisURL:         envOrDefault("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:        envOrDefault("JWT_SECRET", "change-me-in-production"),
		ServerAddr:       envOrDefault("SERVER_ADDR", ":8080"),
		LiveKitURL:       os.Getenv("LIVEKIT_URL"),
		LiveKitAPIKey:    os.Getenv("LIVEKIT_API_KEY"),
		LiveKitAPISecret: os.Getenv("LIVEKIT_API_SECRET"),
		MinIOEndpoint:    os.Getenv("MINIO_ENDPOINT"),
		MinIOAccessKey:   os.Getenv("MINIO_ACCESS_KEY"),
		MinIOSecretKey:   os.Getenv("MINIO_SECRET_KEY"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
