package config

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port string
	Env  string // development, production

	// Database
	DatabaseURL string

	// File paths to 32-byte binary key files.
	SessionSecretFile         string
	SettingsEncryptionKeyFile string
	EmailHMACKeyFile          string

	// Decoded key bytes — populated during Validate(), never set from env directly.
	SessionSecret         []byte
	SettingsEncryptionKey []byte
	EmailHMACKey          []byte

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
}

func Load() (*Config, error) {
	// Load .env file if it exists (don't error if missing)
	_ = godotenv.Load()

	cfg := &Config{}

	// Define flags with env var fallbacks
	flag.StringVar(&cfg.Port, "port", getEnv("PORT", "8080"), "Server port")
	flag.StringVar(&cfg.Env, "env", getEnv("ENV", "development"), "Environment (development, production)")
	flag.StringVar(&cfg.DatabaseURL, "database-url", getEnv("DATABASE_URL", ""), "PostgreSQL connection string")

	cfg.SessionSecretFile = mustEnv("SESSION_SECRET_FILE")
	cfg.SettingsEncryptionKeyFile = mustEnv("SETTINGS_ENCRYPTION_KEY_FILE")
	cfg.EmailHMACKeyFile = mustEnv("EMAIL_HMAC_KEY_FILE")
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

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	sessionKey, err := loadKeyFile(c.SessionSecretFile, "SESSION_SECRET_FILE")
	if err != nil {
		return err
	}
	c.SessionSecret = sessionKey

	key, err := loadKeyFile(c.SettingsEncryptionKeyFile, "SETTINGS_ENCRYPTION_KEY_FILE")
	if err != nil {
		return err
	}
	c.SettingsEncryptionKey = key

	hmacKey, err := loadKeyFile(c.EmailHMACKeyFile, "EMAIL_HMAC_KEY_FILE")
	if err != nil {
		return err
	}
	c.EmailHMACKey = hmacKey

	return nil
}

// loadKeyFile reads a binary key file and returns its contents.
// The file must contain exactly 32 bytes.
func loadKeyFile(path, envVar string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("%s is required", envVar)
	}
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading key file for %s (%q): %w", envVar, path, err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("%s key file must contain exactly 32 bytes (got %d)", envVar, len(key))
	}
	return key, nil
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
