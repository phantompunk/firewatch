package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	dbpkg "github.com/firewatch/internal/db"
	"github.com/firewatch/internal/model"
)

type SchemaStore struct {
	q  *dbpkg.Queries
	db *sql.DB
}

func NewSchemaStore(db *sql.DB) *SchemaStore {
	return &SchemaStore{q: dbpkg.New(db), db: db}
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
	raw, err := s.q.GetReportSchema(ctx, fastBoolConv(live))
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
	raw, err := json.Marshal(schema)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := s.q.WithTx(tx)
	if err := q.DeleteDraftSchemas(ctx); err != nil {
		return fmt.Errorf("delete drafts: %w", err)
	}

	err = q.InsertDraftSchema(ctx, dbpkg.InsertDraftSchemaParams{
		Version:    int64(schema.SchemaVersion),
		SchemaData: json.RawMessage(raw),
		UpdatedBy:  sql.NullString{String: updatedBy, Valid: updatedBy != ""},
	})
	if err != nil {
		return fmt.Errorf("insert draft: %w", err)
	}
	return tx.Commit()
}

// PromoteDraft atomically sets the latest draft as live, then seeds a new
// draft from the published schema so the editor always starts from the
// current live state.
func (s *SchemaStore) PromoteDraft(ctx context.Context, updatedBy string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	qtx := s.q.WithTx(tx)
	if err := qtx.DemoteLiveSchemas(ctx); err != nil {
		return err
	}

	if err := qtx.PromoteLatestDraft(ctx, sql.NullString{String: updatedBy, Valid: updatedBy != ""}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
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

// RevertDraftToLive overwrites the current draft with the live schema,
// effectively discarding any unpublished changes.
func (s *SchemaStore) RevertDraftToLive(ctx context.Context, updatedBy string) error {
	live, err := s.load(ctx, true)
	if err != nil {
		return fmt.Errorf("revert draft to live: %w", err)
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

	// Insert draft row.
	if err := s.q.InsertDraftSchema(ctx, dbpkg.InsertDraftSchemaParams{
		Version: int64(schema.SchemaVersion),
		SchemaData:  json.RawMessage(raw),
		UpdatedBy:  sql.NullString{String: "admin", Valid: true},
	}); err != nil {
		return err
	}

	// Insert live row.
	if err := s.q.PromoteLatestDraft(ctx, sql.NullString{String: "admin", Valid: true}); err != nil {
		return err
	}

	return s.q.InsertDraftSchema(ctx, dbpkg.InsertDraftSchemaParams{
		Version: int64(schema.SchemaVersion),
		SchemaData:  json.RawMessage(raw),
		UpdatedBy:  sql.NullString{String: "admin", Valid: true},
	})
}

func fastBoolConv(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
