package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	dbpkg "github.com/firewatch/internal/db"
)

// ReportStore records anonymous submission events and provides aggregate stats.
// No report content or submitter identity is ever stored.
type ReportStore struct {
	q  *dbpkg.Queries
	db *sql.DB
}

func NewReportStore(db *sql.DB) *ReportStore {
	return &ReportStore{q: dbpkg.New(db), db: db}
}

// RecordEvent persists a submission event with only the IDs of fields that had
// non-empty values. No field values or submitter identity are stored.
func (s *ReportStore) RecordEvent(ctx context.Context, filledFieldIDs []string) error {
	if filledFieldIDs == nil {
		filledFieldIDs = []string{}
	}
	raw, err := json.Marshal(filledFieldIDs)
	if err != nil {
		return fmt.Errorf("marshal field ids: %w", err)
	}
	return s.q.InsertReportEvent(ctx, string(raw))
}

// DailyCount is one row from the per-day aggregation query.
type DailyCount struct {
	Day   string
	Count int64
}

// FieldCount is the number of submissions in which a given field was filled.
type FieldCount struct {
	FieldID string
	Count   int64
}

// ReportStats is the aggregated data needed to render the stats page.
type ReportStats struct {
	Total         int64
	ThisMonth     int64
	ThisWeek      int64
	Today         int64
	LastSubmitted string // SQLite datetime string, empty if no events yet
	DailyActivity []DailyCount
	FieldCounts   []FieldCount
}

// Stats queries all aggregate data for the stats page.
func (s *ReportStore) Stats(ctx context.Context) (*ReportStats, error) {
	now := time.Now().UTC()

	total, err := s.q.CountAllReportEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("count all: %w", err)
	}

	// Beginning of current month.
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	thisMonth, err := s.q.CountReportEventsSince(ctx, monthStart.Format("2006-01-02 15:04:05"))
	if err != nil {
		return nil, fmt.Errorf("count month: %w", err)
	}

	// Monday of current week.
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday → 7 so Monday is day 1
	}
	weekStart := now.AddDate(0, 0, -(weekday - 1)).Truncate(24 * time.Hour)
	thisWeek, err := s.q.CountReportEventsSince(ctx, weekStart.Format("2006-01-02 15:04:05"))
	if err != nil {
		return nil, fmt.Errorf("count week: %w", err)
	}

	// Beginning of today.
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	today, err := s.q.CountReportEventsSince(ctx, todayStart.Format("2006-01-02 15:04:05"))
	if err != nil {
		return nil, fmt.Errorf("count today: %w", err)
	}

	// Latest submission timestamp.
	var lastSubmitted string
	if total > 0 {
		lastSubmitted, err = s.q.LatestReportEventTime(ctx)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("latest event: %w", err)
		}
	}

	// Per-day counts for the last 7 days.
	sevenDaysAgo := now.AddDate(0, 0, -6).Truncate(24 * time.Hour)
	rows, err := s.q.ReportEventsByDay(ctx, sevenDaysAgo.Format("2006-01-02 15:04:05"))
	if err != nil {
		return nil, fmt.Errorf("events by day: %w", err)
	}
	daily := make([]DailyCount, len(rows))
	for i, r := range rows {
		day := fmt.Sprintf("%v", r.Day) // Day is interface{} from date() expression
		daily[i] = DailyCount{Day: day, Count: r.Count}
	}

	// Per-field fill counts using json_each (not expressible in sqlc).
	fieldCounts, err := s.fieldFillCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("field fill counts: %w", err)
	}

	return &ReportStats{
		Total:         total,
		ThisMonth:     thisMonth,
		ThisWeek:      thisWeek,
		Today:         today,
		LastSubmitted: lastSubmitted,
		DailyActivity: daily,
		FieldCounts:   fieldCounts,
	}, nil
}

// fieldFillCounts returns how many submissions included each field ID.
func (s *ReportStore) fieldFillCounts(ctx context.Context) ([]FieldCount, error) {
	const q = `
		SELECT f.value AS field_id, COUNT(*) AS fill_count
		FROM report_events, json_each(fields_filled) AS f
		GROUP BY f.value
		ORDER BY fill_count DESC`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FieldCount
	for rows.Next() {
		var fc FieldCount
		if err := rows.Scan(&fc.FieldID, &fc.Count); err != nil {
			return nil, err
		}
		out = append(out, fc)
	}
	return out, rows.Err()
}
