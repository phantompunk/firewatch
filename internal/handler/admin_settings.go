package handler

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/firewatch/internal/mailer"
	appmw "github.com/firewatch/internal/middleware"
	"github.com/firewatch/internal/model"
)

type adminSettingsPageData struct {
	*model.AppSettings
	IsSuperAdmin bool
	SMTPPassSet  bool
}

// appSettingsResponse is the JSON shape returned by the Get endpoint.
// SMTPPass is replaced by SMTPPassSet so the password never leaves the server.
type appSettingsResponse struct {
	DestinationEmail      string `json:"destinationEmail"`
	EmailSubjectTemplate  string `json:"emailSubjectTemplate"`
	SMTPHost              string `json:"smtpHost"`
	SMTPPort              int    `json:"smtpPort"`
	SMTPUser              string `json:"smtpUser"`
	SMTPPassSet           bool   `json:"smtpPassSet"`
	SMTPFromAddress       string `json:"smtpFromAddress"`
	SMTPFromName          string `json:"smtpFromName"`
	ReportRetentionPolicy string `json:"reportRetentionPolicy"`
	MaintenanceMode       bool   `json:"maintenanceMode"`
	PGPKey                string `json:"pgpKey"`
	SMTPVerified          bool   `json:"smtpVerified"`
	SMTPError             string `json:"smtpError"`
	PGPVerified           bool   `json:"pgpVerified"`
	PGPError              string `json:"pgpError"`
}

func settingsToResponse(s *model.AppSettings) appSettingsResponse {
	return appSettingsResponse{
		DestinationEmail:      s.DestinationEmail,
		EmailSubjectTemplate:  s.EmailSubjectTemplate,
		SMTPHost:              s.SMTPHost,
		SMTPPort:              s.SMTPPort,
		SMTPUser:              s.SMTPUser,
		SMTPPassSet:           s.SMTPPass != "",
		SMTPFromAddress:       s.SMTPFromAddress,
		SMTPFromName:          s.SMTPFromName,
		ReportRetentionPolicy: s.ReportRetentionPolicy,
		MaintenanceMode:       s.MaintenanceMode,
		PGPKey:                s.PGPKey,
		SMTPVerified:          s.SMTPVerified,
		SMTPError:             s.SMTPError,
		PGPVerified:           s.PGPVerified,
		PGPError:              s.PGPError,
	}
}

type settingsStore interface {
	Load(ctx context.Context) (*model.AppSettings, error)
	Save(ctx context.Context, settings *model.AppSettings) error
}

// SettingsHandler handles admin settings views and API.
type SettingsHandler struct {
	BaseHandler
	settings  settingsStore
	mailer    mailer.PingSender
	templates *template.Template
}

func NewSettingsHandler(logger *slog.Logger, settings settingsStore, m mailer.PingSender, tmpl *template.Template) *SettingsHandler {
	return &SettingsHandler{BaseHandler: BaseHandler{logger: logger}, settings: settings, mailer: m, templates: tmpl}
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
		SMTPPassSet:  s.SMTPPass != "",
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

	// s.SMTPPass = "********"
	if err = h.writeJSON(w, http.StatusOK, settingsToResponse(s), nil); err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// verificationResult is the JSON shape returned by Update and Apply.
type verificationResult struct {
	SMTPVerified bool   `json:"smtpVerified"`
	SMTPError    string `json:"smtpError"`
	PGPVerified  bool   `json:"pgpVerified"`
	PGPError     string `json:"pgpError"`
}

// verifyAndPersist runs SMTP and PGP verification against s, persists the
// updated flags, and reconfigures the live mailer.
func (h *SettingsHandler) verifyAndPersist(ctx context.Context, s *model.AppSettings) {
	tmp := mailer.New(mailer.NewConfigFromSettings(s))

	if err := tmp.Ping(); err != nil {
		s.SMTPVerified = false
		s.SMTPError = err.Error()
	} else {
		s.SMTPVerified = true
		s.SMTPError = ""
	}

	if err := tmp.CanEncrypt(); err != nil {
		s.PGPVerified = false
		s.PGPError = err.Error()
	} else {
		s.PGPVerified = true
		s.PGPError = ""
	}

	if err := h.settings.Save(ctx, s); err != nil {
		slog.Error("settings: failed to persist verification state", "err", err)
	}

	if !s.SMTPVerified || !s.PGPVerified {
		slog.Warn("settings: auto-maintenance active",
			"smtpVerified", s.SMTPVerified,
			"smtpError", s.SMTPError,
			"pgpVerified", s.PGPVerified,
			"pgpError", s.PGPError,
		)
	}

	h.mailer.Reconfigure(mailer.NewConfigFromSettings(s))
}

// Update saves updated settings, runs verification, and returns the result as JSON.
func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	s := &model.AppSettings{}
	if err := h.readJSON(w, r, &s); err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}

	if isPrivatePGPKey(s.PGPKey) {
		http.Error(w, "PGP private keys are not accepted — paste the public key only", http.StatusBadRequest)
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

	// Save first so the password is persisted before verification.
	if err := h.settings.Save(r.Context(), s); err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}

	h.verifyAndPersist(r.Context(), s)

	result := verificationResult{
		SMTPVerified: s.SMTPVerified,
		SMTPError:    s.SMTPError,
		PGPVerified:  s.PGPVerified,
		PGPError:     s.PGPError,
	}
	if err := h.writeJSON(w, http.StatusOK, result, nil); err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// Apply re-initialises the mailer with current settings and runs verification.
func (h *SettingsHandler) Apply(w http.ResponseWriter, r *http.Request) {
	s, err := h.settings.Load(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.verifyAndPersist(r.Context(), s)

	result := verificationResult{
		SMTPVerified: s.SMTPVerified,
		SMTPError:    s.SMTPError,
		PGPVerified:  s.PGPVerified,
		PGPError:     s.PGPError,
	}
	if err := h.writeJSON(w, http.StatusOK, result, nil); err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// TestEmail sends a test ping using the saved settings.
// No credentials are accepted from the client — the stored values are always used.
func (h *SettingsHandler) TestEmail(w http.ResponseWriter, r *http.Request) {
	s, err := h.settings.Load(r.Context())
	if err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}
	tmp := mailer.New(mailer.NewConfigFromSettings(s))
	if err := tmp.Ping(); err != nil {
		h.logger.Error("settings: test ping failed", "err", err)
		http.Error(w, "Send failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// isPrivatePGPKey reports whether the given string looks like a PGP private key.
// Both modern and legacy (SECRET KEY) armour headers are checked.
func isPrivatePGPKey(key string) bool {
	return strings.Contains(key, "-----BEGIN PGP PRIVATE KEY BLOCK-----") ||
		strings.Contains(key, "-----BEGIN PGP SECRET KEY BLOCK-----")
}
