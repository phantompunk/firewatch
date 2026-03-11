package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// DeliveryStore records email and submission delivery outcomes.
type DeliveryStore struct {
	db *sql.DB
}

func NewDeliveryStore(db *sql.DB) *DeliveryStore {
	return &DeliveryStore{db: db}
}

// Record inserts a delivery event. Errors are logged, not returned, so
// recording failures never affect the caller's critical path.
//
// kind: "email" | "submission"
// status: "ok" | "error"
func (s *DeliveryStore) Record(ctx context.Context, kind, status string) {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO delivery_log (kind, status) VALUES (?, ?)`, kind, status)
	if err != nil {
		slog.Error("delivery_log: failed to record", "kind", kind, "status", status, "err", err)
	}
}

// DeliveryStats holds 24-hour counts broken down by kind and status.
type DeliveryStats struct {
	EmailOK     int64
	EmailError  int64
	SubmitOK    int64
	SubmitError int64
}

// Stats24h returns delivery and submission counts for the last 24 hours.
func (s *DeliveryStore) Stats24h(ctx context.Context) (*DeliveryStats, error) {
	since := time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	rows, err := s.db.QueryContext(ctx,
		`SELECT kind, status, COUNT(*) FROM delivery_log WHERE created_at >= ? GROUP BY kind, status`,
		since)
	if err != nil {
		return nil, fmt.Errorf("delivery stats: %w", err)
	}
	defer rows.Close()

	var out DeliveryStats
	for rows.Next() {
		var kind, status string
		var count int64
		if err := rows.Scan(&kind, &status, &count); err != nil {
			return nil, fmt.Errorf("delivery stats scan: %w", err)
		}
		switch {
		case kind == "email" && status == "ok":
			out.EmailOK = count
		case kind == "email" && status == "error":
			out.EmailError = count
		case kind == "submission" && status == "ok":
			out.SubmitOK = count
		case kind == "submission" && status == "error":
			out.SubmitError = count
		}
	}
	return &out, rows.Err()
}
