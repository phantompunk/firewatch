package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"log/slog"
	"time"

	dbpkg "github.com/firewatch/internal/db"
)

const sessionTTL = 4 * time.Hour

type SessionStore struct {
	q *dbpkg.Queries
}

func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{q: dbpkg.New(db)}
}

// Create inserts a new session and returns its ID.
func (s *SessionStore) Create(ctx context.Context, userID string) (string, error) {
	id := newToken()
	expiresAt := time.Now().Add(sessionTTL).UTC()
	slog.Info("creating session", "session_id", id, "user_id", userID, "expires_at", expiresAt.Format(time.RFC3339))
	err := s.q.CreateSession(ctx, dbpkg.CreateSessionParams{
		ID:        id,
		UserID:    userID,
		ExpiresAt: expiresAt.UTC().Format("2006-01-02 15:04:05"),
	})
	return id, err
}

// GetUserID validates the session and returns the associated user ID.
// Returns an error if the session does not exist or is expired.
func (s *SessionStore) GetUserID(ctx context.Context, sessionID string) (string, error) {
	return s.q.GetSessionUserID(ctx, sessionID)
}

// DeleteAllByUserID removes all sessions for a user (used on logout / password change).
func (s *SessionStore) DeleteAllByUserID(ctx context.Context, userID string) error {
	return s.q.DeleteSessionsByUserID(ctx, userID)
}

// DeleteExpired removes expired sessions.
func (s *SessionStore) DeleteExpired(ctx context.Context) error {
	return s.q.DeleteExpiredSessions(ctx)
}

func newToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

