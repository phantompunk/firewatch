package store

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/firewatch/internal/crypto"
	dbpkg "github.com/firewatch/internal/db"
	"github.com/firewatch/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

type UserStore struct {
	q       *dbpkg.Queries
	pool    *pgxpool.Pool
	crypter *crypto.Crypter
	hmacKey []byte
}

func NewUserStore(pool *pgxpool.Pool, crypter *crypto.Crypter, hmacKey []byte) *UserStore {
	return &UserStore{q: dbpkg.New(pool), pool: pool, crypter: crypter, hmacKey: hmacKey}
}

func (s *UserStore) CountAll(ctx context.Context) (int, error) {
	n, err := s.q.CountAdminUsers(ctx)
	return int(n), err
}

// Create inserts a new admin user, encrypting the email and computing its HMAC.
func (s *UserStore) Create(ctx context.Context, id, username, email, passwordHash, role string) error {
	emailEnc, err := s.crypter.Encrypt([]byte(email))
	if err != nil {
		return fmt.Errorf("encrypt email: %w", err)
	}
	emailHMAC := crypto.EmailHMAC(s.hmacKey, email)
	return s.q.CreateAdminUser(ctx, dbpkg.CreateAdminUserParams{
		ID:             id,
		Username:       username,
		EmailHmac:      emailHMAC,
		EmailEncrypted: emailEnc,
		PasswordHash:   passwordHash,
		Role:           role,
	})
}

// GetByEmailHMAC looks up a user by the HMAC of their email address.
// Returns the user model and the password hash for verification.
func (s *UserStore) GetByEmailHMAC(ctx context.Context, email string) (*model.AdminUser, string, error) {
	h := crypto.EmailHMAC(s.hmacKey, email)
	row, err := s.q.GetAdminUserByEmailHMAC(ctx, h)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", fmt.Errorf("get user by email hmac: %w", err)
	}
	u := &model.AdminUser{
		ID:          row.ID,
		Username:    row.Username,
		Role:        model.Role(row.Role),
		Status:      model.Status(row.Status),
		CreatedAt:   row.CreatedAt.Time,
		LastLoginAt: pgtimePtr(row.LastLoginAt),
	}
	return u, row.PasswordHash, nil
}

// GetByUsername looks up a user by username.
// Returns the user model and the password hash for verification.
func (s *UserStore) GetByUsername(ctx context.Context, username string) (*model.AdminUser, string, error) {
	row, err := s.q.GetAdminUserByUsername(ctx, username)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", fmt.Errorf("get user by username: %w", err)
	}
	u := &model.AdminUser{
		ID:          row.ID,
		Username:    row.Username,
		Role:        model.Role(row.Role),
		Status:      model.Status(row.Status),
		CreatedAt:   row.CreatedAt.Time,
		LastLoginAt: pgtimePtr(row.LastLoginAt),
	}
	return u, row.PasswordHash, nil
}

func (s *UserStore) GetByID(ctx context.Context, id string) (*model.AdminUser, error) {
	row, err := s.q.GetAdminUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &model.AdminUser{
		ID:          row.ID,
		Username:    row.Username,
		Role:        model.Role(row.Role),
		Status:      model.Status(row.Status),
		CreatedAt:   row.CreatedAt.Time,
		LastLoginAt: pgtimePtr(row.LastLoginAt),
	}, nil
}

func (s *UserStore) ListAll(ctx context.Context) ([]model.AdminUser, error) {
	rows, err := s.q.ListAdminUsers(ctx)
	if err != nil {
		return nil, err
	}
	users := make([]model.AdminUser, len(rows))
	for i, row := range rows {
		users[i] = model.AdminUser{
			ID:          row.ID,
			Username:    row.Username,
			Role:        model.Role(row.Role),
			Status:      model.Status(row.Status),
			CreatedAt:   row.CreatedAt.Time,
			LastLoginAt: pgtimePtr(row.LastLoginAt),
		}
	}
	return users, nil
}

// GetEmailByID decrypts and returns the plaintext email for the given user ID.
// Used by the password-reset flow to send the reset email.
func (s *UserStore) GetEmailByID(ctx context.Context, id string) (string, error) {
	enc, err := s.q.GetAdminUserEmailEncryptedByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get encrypted email: %w", err)
	}
	plain, err := s.crypter.Decrypt(enc)
	if err != nil {
		return "", fmt.Errorf("decrypt email: %w", err)
	}
	return string(plain), nil
}

func (s *UserStore) UpdateRoleAndStatus(ctx context.Context, id string, role model.Role, status model.Status) error {
	return s.q.UpdateAdminUserRoleAndStatus(ctx, dbpkg.UpdateAdminUserRoleAndStatusParams{
		Role:   string(role),
		Status: string(status),
		ID:     id,
	})
}

func (s *UserStore) UpdatePassword(ctx context.Context, id, hash string) error {
	return s.q.UpdateAdminUserPassword(ctx, dbpkg.UpdateAdminUserPasswordParams{
		PasswordHash: hash,
		ID:           id,
	})
}

func (s *UserStore) UpdateLastLogin(ctx context.Context, id string) error {
	return s.q.UpdateAdminUserLastLogin(ctx, id)
}

func (s *UserStore) Delete(ctx context.Context, id string) error {
	superCount, err := s.q.CountActiveSuperAdmins(ctx)
	if err != nil {
		return err
	}
	role, err := s.q.GetAdminUserRoleByID(ctx, id)
	if err != nil {
		return err
	}
	if role == "super_admin" && superCount <= 1 {
		return errLastSuperAdmin
	}
	return s.q.DeleteAdminUser(ctx, id)
}

// CreateInvite stores a hashed invitation token with the email encrypted.
func (s *UserStore) CreateInvite(ctx context.Context, id, email, role, rawToken string) error {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawToken)))
	emailEnc, err := s.crypter.Encrypt([]byte(email))
	if err != nil {
		return fmt.Errorf("encrypt invite email: %w", err)
	}
	return s.q.CreateInvite(ctx, dbpkg.CreateInviteParams{
		ID:             id,
		EmailEncrypted: emailEnc,
		Role:           role,
		TokenHash:      hash,
		ExpiresAt:      pgtype.Timestamptz{Time: time.Now().Add(48 * time.Hour), Valid: true},
	})
}

// GetInviteByToken looks up an active (unused, unexpired) invitation by its raw token.
func (s *UserStore) GetInviteByToken(ctx context.Context, rawToken string) (*model.Invite, error) {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawToken)))
	row, err := s.q.GetInviteByTokenHash(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get invite by token: %w", err)
	}
	emailPlain, err := s.crypter.Decrypt(row.EmailEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt invite email: %w", err)
	}
	return &model.Invite{
		ID:    row.ID,
		Email: string(emailPlain),
		Role:  model.Role(row.Role),
	}, nil
}

// AcceptInvite creates the new admin user and marks the invite as used in one transaction.
func (s *UserStore) AcceptInvite(ctx context.Context, inviteID, userID, username, email, passwordHash, role string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	emailEnc, err := s.crypter.Encrypt([]byte(email))
	if err != nil {
		return fmt.Errorf("encrypt email: %w", err)
	}
	emailHMAC := crypto.EmailHMAC(s.hmacKey, email)

	q := s.q.WithTx(tx)
	if err := q.CreateAdminUser(ctx, dbpkg.CreateAdminUserParams{
		ID:             userID,
		Username:       username,
		EmailHmac:      emailHMAC,
		EmailEncrypted: emailEnc,
		PasswordHash:   passwordHash,
		Role:           role,
	}); err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}
	if err := q.MarkInviteUsed(ctx, inviteID); err != nil {
		return fmt.Errorf("mark invite used: %w", err)
	}
	return tx.Commit(ctx)
}

var errLastSuperAdmin = errStr("cannot delete the last super_admin account")

type errStr string

func (e errStr) Error() string { return string(e) }

func pgtimePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
