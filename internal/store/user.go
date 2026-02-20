package store

import (
	"context"
	"time"

	dbpkg "github.com/firewatch/internal/db"
	"github.com/firewatch/internal/model"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserStore struct {
	q *dbpkg.Queries
}

func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{q: dbpkg.New(pool)}
}

func (s *UserStore) CountAll(ctx context.Context) (int, error) {
	n, err := s.q.CountAdminUsers(ctx)
	return int(n), err
}

func (s *UserStore) Create(ctx context.Context, id, email, passwordHash, role string) error {
	return s.q.CreateAdminUser(ctx, dbpkg.CreateAdminUserParams{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
	})
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*model.AdminUser, string, error) {
	row, err := s.q.GetAdminUserByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}
	u := &model.AdminUser{
		ID:          row.ID,
		Email:       row.Email,
		Role:        model.Role(row.Role),
		Status:      model.Status(row.Status),
		CreatedAt:   row.CreatedAt.Time,
		LastLoginAt: pgtimePtr(row.LastLoginAt),
	}
	return u, row.PasswordHash, nil
}

func (s *UserStore) GetByID(ctx context.Context, id string) (*model.AdminUser, error) {
	row, err := s.q.GetAdminUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &model.AdminUser{
		ID:          row.ID,
		Email:       row.Email,
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
			Email:       row.Email,
			Role:        model.Role(row.Role),
			Status:      model.Status(row.Status),
			CreatedAt:   row.CreatedAt.Time,
			LastLoginAt: pgtimePtr(row.LastLoginAt),
		}
	}
	return users, nil
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

var errLastSuperAdmin = errStr("cannot delete the last super_admin account")

type errStr string

func (e errStr) Error() string { return string(e) }

func pgtimePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
