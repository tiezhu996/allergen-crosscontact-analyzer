package config

import "testing"

func TestLoadRejectsShortJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "short")
	if _, err := Load(); err == nil {
		t.Fatal("Load accepted a JWT secret shorter than 32 bytes")
	}
}

func TestLoadRejectsZeroRateLimit(t *testing.T) {
	t.Setenv("RATE_LIMIT_PER_MINUTE", "0")
	if _, err := Load(); err == nil {
		t.Fatal("Load accepted RATE_LIMIT_PER_MINUTE = 0")
	}
}

func TestLoadRejectsNonAscendingThresholds(t *testing.T) {
	t.Setenv("RISK_THRESHOLD_MEDIUM", "0.9")
	t.Setenv("RISK_THRESHOLD_HIGH", "0.35")
	t.Setenv("RISK_THRESHOLD_CRITICAL", "0.65")
	if _, err := Load(); err == nil {
		t.Fatal("Load accepted non-ascending risk thresholds")
	}
}

func TestLoadRejectsUnsupportedDriver(t *testing.T) {
	t.Setenv("DB_DRIVER", "oracle")
	if _, err := Load(); err == nil {
		t.Fatal("Load accepted an unsupported DB_DRIVER")
	}
}
