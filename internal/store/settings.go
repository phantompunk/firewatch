package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strconv"

	"github.com/firewatch/internal/crypto"
	dbpkg "github.com/firewatch/internal/db"
	"github.com/firewatch/internal/model"
)

type SettingsStore struct {
	q       *dbpkg.Queries
	crypter *crypto.Crypter
}

func NewSettingsStore(db *sql.DB, crypter *crypto.Crypter) *SettingsStore {
	return &SettingsStore{q: dbpkg.New(db), crypter: crypter}
}

// Load decrypts and returns the current settings. Seeds from env vars if no row exists.
func (s *SettingsStore) Load(ctx context.Context) (*model.AppSettings, error) {
	data, err := s.q.GetSettings(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		defaults := settingsFromEnv()
		if saveErr := s.Save(ctx, defaults); saveErr != nil {
			return nil, saveErr
		}
		return defaults, nil
	} else if err != nil {
		return nil, err
	}

	slog.Info("settings: loaded from database")
	plaintext, err := s.crypter.Decrypt(data)
	if err != nil {
		slog.Error("settings: decryption failed", "err", err)
		return nil, err
	}
	var settings model.AppSettings
	if err := json.Unmarshal(plaintext, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

// Save encrypts and persists settings.
func (s *SettingsStore) Save(ctx context.Context, settings *model.AppSettings) error {
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	ciphertext, err := s.crypter.Encrypt(raw)
	if err != nil {
		return err
	}
	return s.q.UpsertSettings(ctx, ciphertext)
}

func settingsFromEnv() *model.AppSettings {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if port == 0 {
		port = 587
	}
	return &model.AppSettings{
		DestinationEmail:      os.Getenv("DESTINATION_EMAIL"),
		EmailSubjectTemplate:  "New Community Report",
		SMTPHost:              os.Getenv("SMTP_HOST"),
		SMTPPort:              port,
		SMTPUser:              os.Getenv("SMTP_USER"),
		SMTPPass:              os.Getenv("SMTP_PASS"),
		SMTPFromAddress:       os.Getenv("SMTP_FROM_ADDRESS"),
		SMTPFromName:          os.Getenv("SMTP_FROM_NAME"),
		ReportRetentionPolicy: "forward-only",
		MaintenanceMode:       true,
	}
}
