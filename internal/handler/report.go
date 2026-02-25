package handler

import (
	"context"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"sort"

	"github.com/firewatch/internal/mailer"
	"github.com/firewatch/internal/middleware"
	"github.com/firewatch/internal/model"
)

type schemaLoader interface {
	LiveSchema(ctx context.Context) (*model.ReportSchema, error)
}

// ReportHandler handles the public report form and submission.
type ReportHandler struct {
	BaseHandler
	schemas   schemaLoader
	sessions  middleware.SessionReader
	mailer    *mailer.Mailer
	templates *template.Template
}

type reportFormData struct {
	Page        model.PageLocale
	Fields      []reportFieldView
	Languages   []model.LangInfo
	CurrentLang string
	IsAdmin     bool
}

type reportFieldView struct {
	ID          string
	Type        string
	Required    bool
	Options     []string
	Label       string
	Description string
	Placeholder string
}

func NewReportHandler(logger *slog.Logger, schemas schemaLoader, sessions middleware.SessionReader, m *mailer.Mailer, tmpl *template.Template) *ReportHandler {
	return &ReportHandler{BaseHandler: BaseHandler{Logger: logger}, schemas: schemas, sessions: sessions, mailer: m, templates: tmpl}
}

// Form renders the public report form.
func (h *ReportHandler) Form(w http.ResponseWriter, r *http.Request) {
	schema, err := h.schemas.LiveSchema(r.Context())
	if err != nil {
		slog.Error("report: failed to load live schema", "err", err)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Resolve language from query param, falling back to schema default.
	lang := r.URL.Query().Get("lang")
	if !containsString(schema.Languages, lang) {
		lang = schema.DefaultLang()
	}

	// Sort fields by per-language display order.
	fields := make([]model.Field, len(schema.Fields))
	copy(fields, schema.Fields)
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].DisplayOrder(lang) < fields[j].DisplayOrder(lang)
	})

	// Build flat field views with resolved locale strings.
	fieldViews := make([]reportFieldView, len(fields))
	for i, f := range fields {
		locale := f.Locale(lang)
		fieldViews[i] = reportFieldView{
			ID:          f.ID,
			Type:        f.Type,
			Required:    f.Required,
			Options:     f.Options,
			Label:       locale.Label,
			Description: locale.Description,
			Placeholder: locale.Placeholder,
		}
	}

	// Resolve enabled languages with names from SupportedLanguages.
	enabledLangs := make([]model.LangInfo, 0, len(schema.Languages))
	for _, info := range model.SupportedLanguages {
		if containsString(schema.Languages, info.Code) {
			enabledLangs = append(enabledLangs, info)
		}
	}

	isAdmin := false
	if cookie, err := r.Cookie(middleware.SessionCookieName); err == nil {
		if _, err := h.sessions.GetUserID(r.Context(), cookie.Value); err == nil {
			isAdmin = true
		}
	}

	data := reportFormData{
		Page:        schema.Page.Locale(lang),
		Fields:      fieldViews,
		Languages:   enabledLangs,
		CurrentLang: lang,
		IsAdmin:     isAdmin,
	}
	if err := h.templates.ExecuteTemplate(w, "report_form.html", data); err != nil {
		slog.Error("report: template error", "err", err)
	}
}

func (h *ReportHandler) Get(w http.ResponseWriter, r *http.Request) {
	schema, err := h.schemas.LiveSchema(r.Context())
	if err != nil {
		h.Logger.Error("report: failed to load live schema", "err", err)
		h.serverErrorResponse(w, r, err)
		return
	}

	err = h.writeJSON(w, http.StatusOK, envelope{"schema": schema}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}
}

func (h *ReportHandler) RedirectToLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin/login", http.StatusFound)
}

// Submit processes an anonymous report submission.
func (h *ReportHandler) Submit(w http.ResponseWriter, r *http.Request) {
	schema, err := h.schemas.LiveSchema(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	var req struct {
		SchemaVersion int               `json:"schemaVersion"`
		Fields        map[string]string `json:"fields"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Validate required fields.
	for _, f := range schema.Fields {
		if f.Required {
			if v := req.Fields[f.ID]; v == "" {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
		}
	}

	// Always use the English email template for admin notifications.
	emailTmpl := schema.EmailTemplates[model.LangEN]
	body := mailer.RenderTemplate(emailTmpl, req.Fields)
	if err := h.mailer.Send("New Community Report", body); err != nil {
		// Log but do not surface to submitter.
		slog.Error("report: smtp send failed", "err", err)
	}

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status":"submitted"}`))
}

// containsString reports whether s is in the slice.
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
