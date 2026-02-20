package handler

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"

	appmw "github.com/firewatch/internal/middleware"
	"github.com/firewatch/internal/model"
)

type schemaDraftStore interface {
	DraftSchema(ctx context.Context) (*model.ReportSchema, error)
	SaveDraft(ctx context.Context, schema *model.ReportSchema, updatedBy string) error
	PromoteDraft(ctx context.Context, updatedBy string) error
}

// AdminReportHandler handles the admin form editor views and API.
type AdminReportHandler struct {
	schemas   schemaDraftStore
	templates *template.Template
}

func NewAdminReportHandler(schemas schemaDraftStore, tmpl *template.Template) *AdminReportHandler {
	return &AdminReportHandler{schemas: schemas, templates: tmpl}
}

// Page renders the admin report editor.
func (h *AdminReportHandler) Page(w http.ResponseWriter, r *http.Request) {
	schema, err := h.schemas.DraftSchema(r.Context())
	if err != nil {
		slog.Error("admin_report: failed to load draft schema", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := h.templates.ExecuteTemplate(w, "admin_report.html", schema); err != nil {
		slog.Error("admin_report: template error", "err", err)
	}
}

// Get returns the current draft schema as JSON.
func (h *AdminReportHandler) Get(w http.ResponseWriter, r *http.Request) {
	// TODO: implement
	w.WriteHeader(http.StatusNotImplemented)
}

// Update saves a draft schema update.
func (h *AdminReportHandler) Update(w http.ResponseWriter, r *http.Request) {
	// TODO: implement
	_ = appmw.UserIDFromContext(r.Context())
	w.WriteHeader(http.StatusNotImplemented)
}

// Apply promotes the draft schema to live.
func (h *AdminReportHandler) Apply(w http.ResponseWriter, r *http.Request) {
	userID := appmw.UserIDFromContext(r.Context())
	if err := h.schemas.PromoteDraft(r.Context(), userID); err != nil {
		slog.Error("admin_report: failed to promote draft", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
