package handler

import (
	"html/template"
	"log/slog"
	"net/http"

	appmw "github.com/firewatch/internal/middleware"
)

// StatsPageData holds mock statistics for the admin stats page.
type StatsPageData struct {
	IsSuperAdmin      bool
	TotalReports      int
	ReportsThisMonth  int
	ReportsThisWeek   int
	ReportsToday      int
	LastSubmission    string
	AvgPerDay         float64
	TopFields         []FieldStat
	RecentActivity    []ActivityEntry
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

// StatsHandler handles the admin stats page.
type StatsHandler struct {
	BaseHandler
	templates *template.Template
}

func NewStatsHandler(logger *slog.Logger, tmpl *template.Template) *StatsHandler {
	return &StatsHandler{BaseHandler: BaseHandler{logger: logger}, templates: tmpl}
}

// Page renders the admin stats page with mock data.
func (h *StatsHandler) Page(w http.ResponseWriter, r *http.Request) {
	data := StatsPageData{
		IsSuperAdmin:     appmw.IsSuperAdmin(r.Context()),
		TotalReports:     284,
		ReportsThisMonth: 47,
		ReportsThisWeek:  12,
		ReportsToday:     3,
		LastSubmission:   "Today at 11:42 AM",
		AvgPerDay:        6.2,
		TopFields: []FieldStat{
			{Name: "Location", Count: 271, Pct: 95},
			{Name: "Incident Type", Count: 261, Pct: 92},
			{Name: "Description", Count: 248, Pct: 87},
			{Name: "Date of Incident", Count: 219, Pct: 77},
			{Name: "Contact Email", Count: 142, Pct: 50},
		},
		RecentActivity: []ActivityEntry{
			{Date: "Feb 22", Count: 3, Weekday: "Sat", BarHeight: 30},
			{Date: "Feb 21", Count: 7, Weekday: "Fri", BarHeight: 70},
			{Date: "Feb 20", Count: 5, Weekday: "Thu", BarHeight: 50},
			{Date: "Feb 19", Count: 9, Weekday: "Wed", BarHeight: 90},
			{Date: "Feb 18", Count: 4, Weekday: "Tue", BarHeight: 40},
			{Date: "Feb 17", Count: 6, Weekday: "Mon", BarHeight: 60},
			{Date: "Feb 16", Count: 2, Weekday: "Sun", BarHeight: 20},
		},
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "admin_stats.html", data); err != nil {
		slog.Error("stats: template error", "err", err)
	}
}
