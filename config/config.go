package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration.
type Config struct {
	// Database
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// JWT
	JWTSecret      string
	JWTExpiryHours int

	// MinIO
	MinioEndpoint string
	MinioUser     string
	MinioPassword string
	MinioBucket   string
	MinioUseSSL   bool

	// Server
	Port        int
	MaxBodySize int64
	AppEnv      string

	// WebSocket
	AllowedOrigins    []string
	WSReadBufferSize  int
	WSWriteBufferSize int
	WSReadLimit       int64

	// Rate Limiting
	RateLimitRequests int
	RateLimitWindow   time.Duration
}

// Load reads configuration from environment variables and returns a validated Config.
func Load() (*Config, error) {
	cfg := &Config{
		// Database
		DBHost:     GetEnv("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 5432),
		DBUser:     GetEnv("DB_USER", "postgres"),
		DBPassword: GetEnv("DB_PASSWORD", "postgres123"),
		DBName:     GetEnv("DB_NAME", "sandbox_api"),
		DBSSLMode:  GetEnv("DB_SSLMODE", "disable"),

		// JWT
		JWTExpiryHours: getEnvInt("JWT_EXPIRY_HOURS", 24),

		// MinIO
		MinioEndpoint: GetEnv("MINIO_ENDPOINT", "minio:9000"),
		MinioUser:     GetEnv("MINIO_ROOT_USER", "minioadmin"),
		MinioPassword: GetEnv("MINIO_ROOT_PASSWORD", "minioadmin123"),
		MinioBucket:   GetEnv("MINIO_BUCKET", "user-uploads"),
		MinioUseSSL:   GetEnv("MINIO_USE_SSL", "false") == "true",

		// Server
		Port:        getEnvInt("PORT", 8080),
		MaxBodySize: int64(getEnvInt("MAX_BODY_SIZE", 1<<20)),
		AppEnv:      GetEnv("APP_ENV", "development"),

		// WebSocket
		WSReadBufferSize:  getEnvInt("WS_READ_BUFFER_SIZE", 1024),
		WSWriteBufferSize: getEnvInt("WS_WRITE_BUFFER_SIZE", 1024),
		WSReadLimit:       int64(getEnvInt("WS_READ_LIMIT", 4096)),

		// Rate Limiting
		RateLimitRequests: getEnvInt("RATE_LIMIT_REQUESTS", 10),
		RateLimitWindow:   time.Duration(getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 60)) * time.Second,
	}

	// JWT secret is required
	jwtSecret, err := RequireEnv("JWT_SECRET")
	if err != nil {
		return nil, err
	}
	cfg.JWTSecret = jwtSecret

	// Allowed origins
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		for _, o := range strings.Split(origins, ",") {
			cfg.AllowedOrigins = append(cfg.AllowedOrigins, strings.TrimSpace(o))
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that all configuration values are valid.
func (c *Config) Validate() error {
	if len(c.JWTSecret) < 16 {
		return fmt.Errorf("JWT_SECRET must be at least 16 characters long")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535")
	}
	if c.DBPort <= 0 || c.DBPort > 65535 {
		return fmt.Errorf("DB_PORT must be between 1 and 65535")
	}
	if c.JWTExpiryHours <= 0 {
		return fmt.Errorf("JWT_EXPIRY_HOURS must be positive")
	}
	if c.MaxBodySize <= 0 {
		return fmt.Errorf("MAX_BODY_SIZE must be positive")
	}
	return nil
}

// IsProduction returns true if the app is running in production mode.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// GetEnv returns the value of an environment variable or a default value.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// RequireEnv returns the value of a required environment variable.
func RequireEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return value, nil
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
