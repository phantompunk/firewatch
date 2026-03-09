package handler

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	appmw "github.com/firewatch/internal/middleware"
	"github.com/firewatch/internal/model"
	"github.com/firewatch/internal/store"
)

// StatsPageData holds statistics for the admin stats page.
type StatsPageData struct {
	IsSuperAdmin     bool
	TotalReports     int
	ReportsThisMonth int
	ReportsThisWeek  int
	ReportsToday     int
	LastSubmission   string
	AvgPerDay        float64
	TopFields        []FieldStat
	RecentActivity   []ActivityEntry
	BusiestDay       string
	MostCompletedField string
	Nonce            string
}

// FieldStat represents how often a field appears in reports.
type FieldStat struct {
	Name  string
	Count int
	Pct   int
}

// ActivityEntry is a row in the recent activity list.
type ActivityEntry struct {
	Date      string
	Count     int
	Weekday   string
	BarHeight int // pixels for CSS bar height
}

type statsDataSource interface {
	Stats(ctx context.Context) (*store.ReportStats, error)
}

type statsSchemaLoader interface {
	LiveSchema(ctx context.Context) (*model.ReportSchema, error)
}

// StatsHandler handles the admin stats page.
type StatsHandler struct {
	BaseHandler
	templates *template.Template
	events    statsDataSource
	schemas   statsSchemaLoader
}

func NewStatsHandler(logger *slog.Logger, events statsDataSource, schemas statsSchemaLoader, tmpl *template.Template) *StatsHandler {
	return &StatsHandler{BaseHandler: BaseHandler{logger: logger}, templates: tmpl, events: events, schemas: schemas}
}

// Page renders the admin stats page with real data.
func (h *StatsHandler) Page(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.events.Stats(ctx)
	if err != nil {
		slog.Error("stats: failed to load stats", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	schema, err := h.schemas.LiveSchema(ctx)
	if err != nil {
		slog.Error("stats: failed to load schema", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := StatsPageData{
		IsSuperAdmin:     appmw.IsSuperAdmin(ctx),
		Nonce:            appmw.NonceFromContext(ctx),
		TotalReports:     int(stats.Total),
		ReportsThisMonth: int(stats.ThisMonth),
		ReportsThisWeek:  int(stats.ThisWeek),
		ReportsToday:     int(stats.Today),
		LastSubmission:   formatLastSubmission(stats.LastSubmitted),
		AvgPerDay:        avgPerDay(stats.Total, stats.LastSubmitted),
		TopFields:        buildFieldStats(stats.FieldCounts, stats.Total, schema),
		RecentActivity:   buildRecentActivity(stats.DailyActivity),
	}

	data.BusiestDay, data.MostCompletedField = buildSummaryExtras(data.RecentActivity, data.TopFields)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "admin_stats.html", data); err != nil {
		slog.Error("stats: template error", "err", err)
	}
}

// formatLastSubmission converts a SQLite datetime string to a human-readable label.
func formatLastSubmission(raw string) string {
	if raw == "" {
		return "No submissions yet"
	}
	// SQLite stores as "2006-01-02 15:04:05" or RFC3339 variant.
	for _, layout := range []string{"2006-01-02T15:04:05Z", "2006-01-02 15:04:05", time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			now := time.Now().UTC()
			if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
				return "Today at " + t.Format("3:04 PM")
			}
			yesterday := now.AddDate(0, 0, -1)
			if t.Year() == yesterday.Year() && t.YearDay() == yesterday.YearDay() {
				return "Yesterday at " + t.Format("3:04 PM")
			}
			return t.Format("Jan 2 at 3:04 PM")
		}
	}
	return raw
}

// avgPerDay computes average submissions per day since the first submission.
func avgPerDay(total int64, lastSubmitted string) float64 {
	if total == 0 || lastSubmitted == "" {
		return 0
	}
	// Use total / 30 as a simple rolling average over 30 days.
	// If we have fewer days of data that's still a reasonable estimate.
	return float64(total) / 30.0
}

// buildFieldStats maps field IDs to their schema labels and computes percentages.
func buildFieldStats(counts []store.FieldCount, total int64, schema *model.ReportSchema) []FieldStat {
	if total == 0 {
		return nil
	}
	nameByID := make(map[string]string, len(schema.Fields))
	for _, f := range schema.Fields {
		label := f.Locale(schema.DefaultLang()).Label
		if label == "" {
			label = f.ID
		}
		nameByID[f.ID] = label
	}

	stats := make([]FieldStat, 0, len(counts))
	for _, fc := range counts {
		name := nameByID[fc.FieldID]
		if name == "" {
			name = fc.FieldID
		}
		pct := int(fc.Count * 100 / total)
		stats = append(stats, FieldStat{Name: name, Count: int(fc.Count), Pct: pct})
	}
	return stats
}

// buildRecentActivity generates 7 ActivityEntry rows (one per day, oldest→newest).
// Days with no submissions get Count=0.
func buildRecentActivity(daily []store.DailyCount) []ActivityEntry {
	// Index DB results by date string ("2006-01-02").
	byDay := make(map[string]int64, len(daily))
	var maxCount int64
	for _, d := range daily {
		byDay[d.Day] = d.Count
		if d.Count > maxCount {
			maxCount = d.Count
		}
	}

	now := time.Now().UTC()
	entries := make([]ActivityEntry, 7)
	for i := 0; i < 7; i++ {
		day := now.AddDate(0, 0, -(6 - i))
		key := day.Format("2006-01-02")
		count := byDay[key]
		barH := 0
		if maxCount > 0 {
			barH = int(count * 90 / maxCount)
			if barH == 0 && count > 0 {
				barH = 4 // minimum visible bar
			}
		}
		entries[i] = ActivityEntry{
			Date:      day.Format("Jan 2"),
			Count:     int(count),
			Weekday:   day.Format("Mon"),
			BarHeight: barH,
		}
	}
	return entries
}

// buildSummaryExtras returns the busiest day label and most-completed field label.
func buildSummaryExtras(activity []ActivityEntry, fields []FieldStat) (busiestDay string, mostCompleted string) {
	busiestDay = "—"
	var maxCount int
	for _, a := range activity {
		if a.Count > maxCount {
			maxCount = a.Count
			busiestDay = fmt.Sprintf("%s (%d reports)", a.Weekday, a.Count)
		}
	}
	if maxCount == 0 {
		busiestDay = "—"
	}

	mostCompleted = "—"
	if len(fields) > 0 {
		mostCompleted = fmt.Sprintf("%s (%d%%)", fields[0].Name, fields[0].Pct)
	}
	return
}

