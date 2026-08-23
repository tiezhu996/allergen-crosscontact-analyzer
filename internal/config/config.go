package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Thresholds struct {
	Medium   float64
	High     float64
	Critical float64
	Version  string
}

type Config struct {
	Port                string
	DBDriver            string
	DBDSN               string
	DBAutoMigrate       bool
	JWTSecret           string
	JWTExpiry           time.Duration
	CORSOrigins         []string
	MaxPropagationDepth int
	Thresholds          Thresholds
	RateLimitPerMinute  int
	LogLevel            string
}

func Load() (Config, error) {
	cfg := Config{
		Port:        env("PORT", "8080"),
		DBDriver:    strings.ToLower(env("DB_DRIVER", "postgres")),
		DBDSN:       env("DB_DSN", "host=127.0.0.1 user=allergen_app password=allergen_local_password dbname=allergen_crosscontact port=57525 sslmode=disable TimeZone=Asia/Shanghai"),
		JWTSecret:   env("JWT_SECRET", "local-allergen-secret-at-least-32-bytes"),
		CORSOrigins: splitCSV(env("CORS_ORIGINS", "http://localhost:18525,http://127.0.0.1:18525")),
		LogLevel:    env("LOG_LEVEL", "info"),
	}
	var err error
	if cfg.DBAutoMigrate, err = envBool("DB_AUTO_MIGRATE", true); err != nil {
		return Config{}, err
	}
	if cfg.JWTExpiry, err = envDurationHours("JWT_EXPIRY_HOURS", 12); err != nil {
		return Config{}, err
	}
	if cfg.MaxPropagationDepth, err = envInt("MAX_PROPAGATION_DEPTH", 12); err != nil {
		return Config{}, err
	}
	if cfg.RateLimitPerMinute, err = envInt("RATE_LIMIT_PER_MINUTE", 240); err != nil {
		return Config{}, err
	}
	if cfg.Thresholds.Medium, err = envFloat("RISK_THRESHOLD_MEDIUM", 0.12); err != nil {
		return Config{}, err
	}
	if cfg.Thresholds.High, err = envFloat("RISK_THRESHOLD_HIGH", 0.35); err != nil {
		return Config{}, err
	}
	if cfg.Thresholds.Critical, err = envFloat("RISK_THRESHOLD_CRITICAL", 0.65); err != nil {
		return Config{}, err
	}
	cfg.Thresholds.Version = env("RISK_THRESHOLD_VERSION", "2026.1")
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.DBDriver != "postgres" && c.DBDriver != "sqlite" {
		return fmt.Errorf("unsupported DB_DRIVER %q", c.DBDriver)
	}
	if c.DBDSN == "" {
		return fmt.Errorf("DB_DSN is required")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}
	if c.MaxPropagationDepth < 2 || c.MaxPropagationDepth > 64 {
		return fmt.Errorf("MAX_PROPAGATION_DEPTH must be between 2 and 64")
	}
	if !(0 < c.Thresholds.Medium && c.Thresholds.Medium < c.Thresholds.High && c.Thresholds.High < c.Thresholds.Critical && c.Thresholds.Critical <= 1) {
		return fmt.Errorf("risk thresholds must be ascending values in (0,1]")
	}
	if c.RateLimitPerMinute < 10 {
		return fmt.Errorf("RATE_LIMIT_PER_MINUTE must be at least 10")
	}
	return nil
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func envBool(key string, fallback bool) (bool, error) {
	value := env(key, strconv.FormatBool(fallback))
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func envInt(key string, fallback int) (int, error) {
	value := env(key, strconv.Itoa(fallback))
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func envFloat(key string, fallback float64) (float64, error) {
	value := env(key, strconv.FormatFloat(fallback, 'f', -1, 64))
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func envDurationHours(key string, fallback int) (time.Duration, error) {
	hours, err := envInt(key, fallback)
	if err != nil {
		return 0, err
	}
	if hours < 1 || hours > 168 {
		return 0, fmt.Errorf("%s must be between 1 and 168", key)
	}
	return time.Duration(hours) * time.Hour, nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if clean := strings.TrimSpace(part); clean != "" {
			result = append(result, clean)
		}
	}
	return result
}
