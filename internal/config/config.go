package config

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port string
	Env  string // development, production

	// Database
	DatabaseURL string

	// Security
	SessionSecret         string
	SettingsEncryptionKey string
	EmailHMACKey          string

	// SMTP
	SMTPHost              string
	SMTPPort              string
	SMTPUser              string
	SMTPPass              string
	SMTPFromEmail         string
	SMTPFromName          string
	ReportRetentionPolicy string
	DestinationEmail      string

	AdminInviteBaseURL string

	SecureCookies bool
	Cors          struct {
		TrustedOrigins []string
	}
}

func Load() (*Config, error) {
	// Load .env file if it exists (don't error if missing)
	_ = godotenv.Load()

	cfg := &Config{}

	// Define flags with env var fallbacks
	flag.StringVar(&cfg.Port, "port", getEnv("PORT", "8080"), "Server port")
	flag.StringVar(&cfg.Env, "env", getEnv("ENV", "development"), "Environment (development, production)")
	flag.StringVar(&cfg.DatabaseURL, "database-url", getEnv("DATABASE_URL", ""), "PostgreSQL connection string")

	cfg.SessionSecret = mustEnv("SESSION_SECRET")
	cfg.SettingsEncryptionKey = mustEnv("SETTINGS_ENCRYPTION_KEY")
	cfg.EmailHMACKey = mustEnv("EMAIL_HMAC_KEY")
	cfg.SMTPHost = getEnv("SMTP_HOST", "")
	cfg.SMTPPort = getEnv("SMTP_PORT", "587")
	cfg.SMTPUser = getEnv("SMTP_USER", "")
	cfg.SMTPPass = getEnv("SMTP_PASS", "")
	cfg.SMTPFromEmail = getEnv("SMTP_FROM_EMAIL", "")
	cfg.SMTPFromName = getEnv("SMTP_FROM_NAME", "")
	cfg.DestinationEmail = getEnv("DESTINATION_EMAIL", "")
	cfg.ReportRetentionPolicy = getEnv("REPORT_RETENTION_POLICY", "30d")
	cfg.AdminInviteBaseURL = getEnv("ADMIN_INVITE_BASE_URL", "")
	cfg.SecureCookies = getEnv("SECURE_COOKIES", "false") == "true"

	flag.Parse()


	// Parse CORS trusted origins from comma-separated env var
	if origins := getEnv("CORS_TRUSTED_ORIGINS", ""); origins != "" {
		for _, origin := range strings.Split(origins, ",") {
			if trimmed := strings.TrimSpace(origin); trimmed != "" {
				cfg.Cors.TrustedOrigins = append(cfg.Cors.TrustedOrigins, trimmed)
			}
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if len(c.SettingsEncryptionKey) < 32 {
		return fmt.Errorf("SETTINGS_ENCRYPTION_KEY must be at least 32 characters")
	}

	if len(c.EmailHMACKey) < 32 {
		return fmt.Errorf("EMAIL_HMAC_KEY must be at least 32 characters")
	}

	if len(c.SessionSecret) < 16 {
		return fmt.Errorf("SESSION_SECRET must be at least 16 characters")
	}
	// if c.EncryptionSalt == "" {
	// 	return fmt.Errorf("ENCRYPTION_SALT is required")
	// }
	// if len(c.EncryptionSalt) < 16 {
	// 	return fmt.Errorf("ENCRYPTION_SALT must be at least 16 characters")
	// }
	return nil
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func mustEnv(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	slog.Error("missing required environment variable", "key", key)
	os.Exit(1)
	return ""
}
