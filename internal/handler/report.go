package handler

import (
	"context"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"

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
	*model.ReportSchema
	IsAdmin bool
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

	isAdmin := false
	if cookie, err := r.Cookie(middleware.SessionCookieName); err == nil {
		if _, err := h.sessions.GetUserID(r.Context(), cookie.Value); err == nil {
			isAdmin = true
		}
	}

	data := reportFormData{ReportSchema: schema, IsAdmin: isAdmin}
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

	body := mailer.RenderTemplate(schema.EmailTemplate, req.Fields)
	if err := h.mailer.Send("New Community Report", body); err != nil {
		// Log but do not surface to submitter.
		slog.Error("report: smtp send failed", "err", err)
	}

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status":"submitted"}`))
}
