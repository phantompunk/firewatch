package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	dbpkg "github.com/firewatch/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sessionTTL = 60 * time.Minute

type SessionStore struct {
	q *dbpkg.Queries
}

func NewSessionStore(pool *pgxpool.Pool) *SessionStore {
	return &SessionStore{q: dbpkg.New(pool)}
}

// Create inserts a new session and returns its ID.
func (s *SessionStore) Create(ctx context.Context, userID string) (string, error) {
	id := newToken()
	expiresAt := pgtype.Timestamptz{Time: time.Now().Add(sessionTTL), Valid: true}
	err := s.q.CreateSession(ctx, dbpkg.CreateSessionParams{
		ID:        id,
		UserID:    userID,
		ExpiresAt: expiresAt,
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

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
