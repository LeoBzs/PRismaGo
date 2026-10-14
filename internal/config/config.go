// Package config loads runtime settings from environment variables.
package config

import (
	"log/slog"
	"os"
	"strings"
	"time"
)

type Config struct {
	Addr               string
	DatabaseURL        string
	CORSAllowedOrigins []string
	LogLevel           slog.Level
	ShutdownTimeout    time.Duration
}

func Load() Config {
	return Config{
		Addr: ":" + getEnv("PORT", "8080"),
		// Default matches docker-compose.yml. Override in any real environment.
		DatabaseURL: getEnv("DATABASE_URL", "postgres://prismago:prismago@localhost:5432/prismago?sslmode=disable"),
		// Browser origins allowed to call the API, comma-separated. Defaults are
		// local Angular dev servers; set the real frontend URL in production.
		CORSAllowedOrigins: splitList(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:4200,http://localhost:4300")),
		LogLevel:           parseLevel(getEnv("LOG_LEVEL", "info")),
		ShutdownTimeout:    10 * time.Second,
	}
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func parseLevel(s string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(s)); err != nil {
		return slog.LevelInfo
	}
	return l
}
