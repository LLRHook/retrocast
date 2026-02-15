// Package config loads server configuration with a three-tier hierarchy:
//
//  1. Environment variables (highest priority, always win)
//  2. Config file values (middle priority)
//  3. Hardcoded defaults (lowest priority)
//
// The config file is optional. It is searched in this order:
//   - Path in $RETROCAST_CONFIG environment variable
//   - ./retrocast.toml (working directory)
//   - /etc/retrocast/config.toml
//
// The config file uses a simple KEY = VALUE format (one per line).
// Lines starting with # are comments. Empty lines are ignored.
// Keys match environment variable names (e.g. DATABASE_URL, JWT_SECRET).
package config

import (
	"bufio"
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
	fileVals := loadConfigFile()

	cfg := &Config{
		DatabaseURL:      resolve("DATABASE_URL", fileVals, ""),
		RedisURL:         resolve("REDIS_URL", fileVals, "redis://localhost:6379"),
		JWTSecret:        resolve("JWT_SECRET", fileVals, ""),
		ServerAddr:       resolve("SERVER_ADDR", fileVals, ":8080"),
		LogLevel:         parseLogLevel(resolve("LOG_LEVEL", fileVals, "")),
		LiveKitURL:       resolve("LIVEKIT_URL", fileVals, ""),
		LiveKitAPIKey:    resolve("LIVEKIT_API_KEY", fileVals, ""),
		LiveKitAPISecret: resolve("LIVEKIT_API_SECRET", fileVals, ""),
		MinIOEndpoint:    resolve("MINIO_ENDPOINT", fileVals, ""),
		MinIOAccessKey:   resolve("MINIO_ACCESS_KEY", fileVals, ""),
		MinIOSecretKey:   resolve("MINIO_SECRET_KEY", fileVals, ""),
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

// resolve returns the value for a config key using the three-tier hierarchy:
// env var → config file → default.
func resolve(key string, fileVals map[string]string, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	if v, ok := fileVals[key]; ok && v != "" {
		return v
	}
	return fallback
}

// loadConfigFile searches for and parses a config file. Returns an empty map
// if no config file is found (this is not an error — the file is optional).
func loadConfigFile() map[string]string {
	paths := configFilePaths()
	for _, p := range paths {
		if vals, err := parseConfigFile(p); err == nil {
			slog.Info("loaded config file", "path", p)
			return vals
		}
	}
	return map[string]string{}
}

// configFilePaths returns the ordered list of config file paths to try.
func configFilePaths() []string {
	var paths []string
	if p := os.Getenv("RETROCAST_CONFIG"); p != "" {
		paths = append(paths, p)
	}
	paths = append(paths, "retrocast.toml", "/etc/retrocast/config.toml")
	return paths
}

// parseConfigFile reads a simple KEY = VALUE config file.
// Lines starting with # or [ are ignored (comments and TOML section headers).
// Empty lines are skipped. Values may optionally be quoted with double quotes.
func parseConfigFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	vals := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line[0] == '#' || line[0] == '[' {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		// Strip surrounding double quotes if present.
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		vals[key] = val
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return vals, nil
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
