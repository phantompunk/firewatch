package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	dbpkg "github.com/firewatch/internal/db"
	"github.com/firewatch/internal/model"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SchemaStore struct {
	q  *dbpkg.Queries
	db *pgxpool.Pool
}

func NewSchemaStore(pool *pgxpool.Pool) *SchemaStore {
	return &SchemaStore{q: dbpkg.New(pool), db: pool}
}

// LiveSchema returns the currently published schema.
func (s *SchemaStore) LiveSchema(ctx context.Context) (*model.ReportSchema, error) {
	return s.load(ctx, true)
}

// DraftSchema returns the current draft schema.
func (s *SchemaStore) DraftSchema(ctx context.Context) (*model.ReportSchema, error) {
	return s.load(ctx, false)
}

func (s *SchemaStore) load(ctx context.Context, live bool) (*model.ReportSchema, error) {
	raw, err := s.q.GetReportSchema(ctx, live)
	if err != nil {
		return nil, err
	}
	var schema model.ReportSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

// SaveDraft persists the draft schema.
func (s *SchemaStore) SaveDraft(ctx context.Context, schema *model.ReportSchema, updatedBy string) error {
	schema.UpdatedAt = time.Now().UTC()
	schema.UpdatedBy = updatedBy

	raw, err := json.Marshal(schema)
	if err != nil {
		return err
	}
	return s.q.InsertReportSchema(ctx, dbpkg.InsertReportSchemaParams{
		Version:   int32(schema.SchemaVersion),
		IsLive:    false,
		Schema:    raw,
		UpdatedAt: pgtype.Timestamptz{Time: schema.UpdatedAt, Valid: true},
		UpdatedBy: pgtype.Text{String: updatedBy, Valid: updatedBy != ""},
	})
}

// PromoteDraft atomically sets the latest draft as live, then seeds a new
// draft from the published schema so the editor always starts from the
// current live state.
func (s *SchemaStore) PromoteDraft(ctx context.Context, updatedBy string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)
	if err := qtx.DemoteLiveSchemas(ctx); err != nil {
		return err
	}
	if err := qtx.PromoteLatestDraft(ctx, pgtype.Text{String: updatedBy, Valid: updatedBy != ""}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// Copy the just-published live schema into a new draft row so the editor
	// opens from the published version rather than a stale older draft.
	live, err := s.load(ctx, true)
	if err != nil {
		return fmt.Errorf("copy live to draft after promote: %w", err)
	}
	return s.SaveDraft(ctx, live, updatedBy)
}

// SeedDefault inserts the default SALUTE schema as both draft and live if the
// table is empty.
func (s *SchemaStore) SeedDefault(ctx context.Context) error {
	count, err := s.q.CountReportSchemas(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	schema := model.DefaultSALUTESchema()
	raw, err := json.Marshal(schema)
	if err != nil {
		return err
	}

	// Insert live row.
	if err := s.q.InsertReportSchemaRow(ctx, dbpkg.InsertReportSchemaRowParams{
		Version: int32(schema.SchemaVersion),
		IsLive:  true,
		Schema:  raw,
	}); err != nil {
		return err
	}

	// Insert draft row.
	return s.q.InsertReportSchemaRow(ctx, dbpkg.InsertReportSchemaRowParams{
		Version: int32(schema.SchemaVersion),
		IsLive:  false,
		Schema:  raw,
	})
}
