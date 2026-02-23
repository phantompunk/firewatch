package handler

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"

	"github.com/firewatch/internal/mailer"
	appmw "github.com/firewatch/internal/middleware"
	"github.com/firewatch/internal/model"
)

type adminSettingsPageData struct {
	*model.AppSettings
	IsSuperAdmin bool
}

type settingsStore interface {
	Load(ctx context.Context) (*model.AppSettings, error)
	Save(ctx context.Context, settings *model.AppSettings) error
}

// SettingsHandler handles admin settings views and API.
type SettingsHandler struct {
	BaseHandler
	settings  settingsStore
	mailer    *mailer.Mailer
	templates *template.Template
}

func NewSettingsHandler(logger *slog.Logger, settings settingsStore, m *mailer.Mailer, tmpl *template.Template) *SettingsHandler {
	return &SettingsHandler{BaseHandler: BaseHandler{Logger: logger}, settings: settings, mailer: m, templates: tmpl}
}

// Page renders the admin settings page.
func (h *SettingsHandler) Page(w http.ResponseWriter, r *http.Request) {
	s, err := h.settings.Load(r.Context())
	if err != nil {
		slog.Error("settings: failed to load", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	data := adminSettingsPageData{
		AppSettings:  s,
		IsSuperAdmin: appmw.IsSuperAdmin(r.Context()),
	}
	if err := h.templates.ExecuteTemplate(w, "admin_settings.html", data); err != nil {
		slog.Error("settings: template error", "err", err)
	}
}

// Get returns the current settings as JSON (with secrets masked).
func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	s, err := h.settings.Load(r.Context())
	if err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}

	err = h.writeJSON(w, http.StatusOK, s, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
		return 
	}
}

// Update saves updated settings.
func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	s := &model.AppSettings{}
	if err := h.readJSON(w, r, &s); err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}

	if s.SMTPPass == "" {
		current, err := h.settings.Load(r.Context())
		if err != nil {
			h.serverErrorResponse(w, r, err)
			return
		}
		s.SMTPPass = current.SMTPPass
	}

	if err := h.settings.Save(r.Context(), s); err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}
}

// Apply re-initialises the mailer with current settings.
func (h *SettingsHandler) Apply(w http.ResponseWriter, r *http.Request) {
	s, err := h.settings.Load(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	h.mailer.Reconfigure(s)
	w.WriteHeader(http.StatusOK)
}

// TestEmail sends a test email to the configured destination.
func (h *SettingsHandler) TestEmail(w http.ResponseWriter, r *http.Request) {
	s, err := h.settings.Load(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	h.mailer.Reconfigure(s)
	if err := h.mailer.Send("Test Email", "This is a test email from Firewatch."); err != nil {
		slog.Error("settings: test email failed", "err", err)
		http.Error(w, "Send failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusOK)
}
