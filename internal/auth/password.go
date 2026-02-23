package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// Hash returns a bcrypt hash of the password.
func Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(b), err
}

// Verify reports whether password matches the stored bcrypt hash.
func Verify(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// NewID generates a random hex ID.
func NewID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateToken returns a 32-byte cryptographically random hex string.
func GenerateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// UserCreator is the minimal interface needed for seeding the first admin.
type UserCreator interface {
	CountAll(ctx context.Context) (int, error)
	Create(ctx context.Context, id, email, passwordHash, role string) error
}

// SeedFirstAdmin creates the initial super_admin account from env vars if the
// admin_users table is empty.
func SeedFirstAdmin(ctx context.Context, users UserCreator) {
	email := os.Getenv("SEED_ADMIN_EMAIL")
	password := os.Getenv("SEED_ADMIN_PASSWORD")
	if email == "" || password == "" {
		return
	}

	count, err := users.CountAll(ctx)
	if err != nil {
		slog.Error("seed: failed to count admin users", "err", err)
		return
	}
	if count > 0 {
		return
	}

	hash, err := Hash(password)
	if err != nil {
		slog.Error("seed: failed to hash password", "err", err)
		return
	}

	if err := users.Create(ctx, NewID(), email, hash, "super_admin"); err != nil {
		slog.Error("seed: failed to create admin user", "err", err)
		return
	}
	slog.Info("seed: created first super_admin", "email", email)
}
