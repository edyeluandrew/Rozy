package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	Env              string
	DatabaseURL      string
	RedisURL         string
	JWTSecret        string
	SMSProvider      string
	OTPExpiry        time.Duration
	JWTExpiry        time.Duration
	AutoMigrate      bool
	UploadDir        string
	MTNWebhookSecret string
	AirtelWebhookSecret string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "rozy-dev-secret-change-me"
	}

	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		Env:         getEnv("ENV", "development"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:   jwtSecret,
		SMSProvider: getEnv("SMS_PROVIDER", "console"),
		OTPExpiry:   5 * time.Minute,
		JWTExpiry:   7 * 24 * time.Hour,
		AutoMigrate:         getEnv("AUTO_MIGRATE", "true") == "true",
		UploadDir:           getEnv("UPLOAD_DIR", "uploads"),
		MTNWebhookSecret:    os.Getenv("MTN_MOMO_WEBHOOK_SECRET"),
		AirtelWebhookSecret: os.Getenv("AIRTEL_WEBHOOK_SECRET"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
