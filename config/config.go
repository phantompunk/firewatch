package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port       string
	StaticDir  string

	// Email
	SMTPHost      string
	SMTPPort      int
	SMTPUser      string
	SMTPPass      string
	RecipientEmail string
	FromEmail     string

	// PGP
	PGPPublicKeyPath string

	// Limits
	RateLimitPerMinute int
	MaxUploadSizeMB    int
}

// LoadEnv loads environment variables from .env file if it exists.
// Call this before Load() to use .env file configuration.
func LoadEnv() {
	// Load .env file if it exists (ignores error if file not found)
	godotenv.Load()
}

func Load() *Config {
	return &Config{
		Port:              getEnv("PORT", "8080"),
		StaticDir:         getEnv("STATIC_DIR", "./static"),

		SMTPHost:          getEnv("SMTP_HOST", ""),
		SMTPPort:          getEnvInt("SMTP_PORT", 587),
		SMTPUser:          getEnv("SMTP_USER", ""),
		SMTPPass:          getEnv("SMTP_PASS", ""),
		RecipientEmail:    getEnv("RECIPIENT_EMAIL", ""),
		FromEmail:         getEnv("FROM_EMAIL", "noreply@firewatch-reports.org"),

		PGPPublicKeyPath:  getEnv("PGP_PUBLIC_KEY_PATH", ""),

		RateLimitPerMinute: getEnvInt("RATE_LIMIT_PER_MINUTE", 10),
		MaxUploadSizeMB:    getEnvInt("MAX_UPLOAD_SIZE_MB", 50),
	}
}

func (c *Config) Validate() error {
	// For MVP, we'll allow running without email config for testing
	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
