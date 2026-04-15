package config

import (
	"testing"
)

func TestGetEnv(t *testing.T) {
	t.Run("returns default when env not set", func(t *testing.T) {
		got := GetEnv("TOTALLY_UNSET_VAR_XYZ", "fallback")
		if got != "fallback" {
			t.Errorf("GetEnv = %q, want %q", got, "fallback")
		}
	})

	t.Run("returns env value when set", func(t *testing.T) {
		t.Setenv("TEST_GET_ENV_KEY", "custom-value")
		got := GetEnv("TEST_GET_ENV_KEY", "fallback")
		if got != "custom-value" {
			t.Errorf("GetEnv = %q, want %q", got, "custom-value")
		}
	})
}

func TestRequireEnv(t *testing.T) {
	t.Run("returns error when env not set", func(t *testing.T) {
		_, err := RequireEnv("TOTALLY_UNSET_REQUIRED_VAR")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns value when set", func(t *testing.T) {
		t.Setenv("TEST_REQUIRE_ENV_KEY", "required-value")
		got, err := RequireEnv("TEST_REQUIRE_ENV_KEY")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "required-value" {
			t.Errorf("RequireEnv = %q, want %q", got, "required-value")
		}
	})
}

func TestConfig_Validate(t *testing.T) {
	validConfig := func() *Config {
		return &Config{
			JWTSecret:      "at-least-sixteen-chars",
			Port:           8080,
			DBPort:         5432,
			JWTExpiryHours: 24,
			MaxBodySize:    1 << 20,
		}
	}

	t.Run("accepts valid config", func(t *testing.T) {
		cfg := validConfig()
		if err := cfg.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects short JWT secret", func(t *testing.T) {
		cfg := validConfig()
		cfg.JWTSecret = "short"
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for short JWT secret")
		}
	})

	t.Run("rejects port zero", func(t *testing.T) {
		cfg := validConfig()
		cfg.Port = 0
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for port 0")
		}
	})

	t.Run("rejects port above 65535", func(t *testing.T) {
		cfg := validConfig()
		cfg.Port = 70000
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for port > 65535")
		}
	})

	t.Run("rejects negative port", func(t *testing.T) {
		cfg := validConfig()
		cfg.Port = -1
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for negative port")
		}
	})

	t.Run("rejects invalid DB port", func(t *testing.T) {
		cfg := validConfig()
		cfg.DBPort = 0
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for DB port 0")
		}
	})

	t.Run("rejects non-positive JWTExpiryHours", func(t *testing.T) {
		cfg := validConfig()
		cfg.JWTExpiryHours = 0
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for zero JWTExpiryHours")
		}
	})

	t.Run("rejects non-positive MaxBodySize", func(t *testing.T) {
		cfg := validConfig()
		cfg.MaxBodySize = 0
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for zero MaxBodySize")
		}
	})
}

func TestConfig_IsProduction(t *testing.T) {
	tests := []struct {
		name   string
		appEnv string
		want   bool
	}{
		{
			name:   "returns true for production",
			appEnv: "production",
			want:   true,
		},
		{
			name:   "returns false for development",
			appEnv: "development",
			want:   false,
		},
		{
			name:   "returns false for empty",
			appEnv: "",
			want:   false,
		},
		{
			name:   "returns false for staging",
			appEnv: "staging",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{AppEnv: tt.appEnv}
			if got := cfg.IsProduction(); got != tt.want {
				t.Errorf("IsProduction() = %v, want %v", got, tt.want)
			}
		})
	}
}
