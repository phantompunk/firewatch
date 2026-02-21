package handler

import (
	"context"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"

	appmw "github.com/firewatch/internal/middleware"
	"github.com/firewatch/internal/model"
)

type adminReportPageData struct {
	model.ReportSchema
	SchemaJSON template.JS
}

type schemaDraftStore interface {
	DraftSchema(ctx context.Context) (*model.ReportSchema, error)
	SaveDraft(ctx context.Context, schema *model.ReportSchema, updatedBy string) error
	PromoteDraft(ctx context.Context, updatedBy string) error
}

// AdminReportHandler handles the admin form editor views and API.
type AdminReportHandler struct {
	BaseHandler
	schemas   schemaDraftStore
	templates *template.Template
}

func NewAdminReportHandler(logger *slog.Logger, schemas schemaDraftStore, tmpl *template.Template) *AdminReportHandler {
	return &AdminReportHandler{BaseHandler: BaseHandler{Logger: logger}, schemas: schemas, templates: tmpl}
}

// Page renders the admin report editor.
func (h *AdminReportHandler) Page(w http.ResponseWriter, r *http.Request) {
	schema, err := h.schemas.DraftSchema(r.Context())
	if err != nil {
		slog.Error("admin_report: failed to load draft schema", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	jsonBytes, _ := json.Marshal(schema)
	data := adminReportPageData{
		ReportSchema: *schema,
		SchemaJSON:   template.JS(jsonBytes),
	}
	if err := h.templates.ExecuteTemplate(w, "admin_report.html", data); err != nil {
		slog.Error("admin_report: template error", "err", err)
	}
}

// Get returns the current draft schema as JSON.
func (h *AdminReportHandler) Get(w http.ResponseWriter, r *http.Request) {
	schema, err := h.schemas.DraftSchema(r.Context())
	if err != nil {
		h.serverErrorResponse(w, r, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = h.writeJSON(w, http.StatusOK, envelope{"schema": schema}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}
}

// Update saves a draft schema update.
func (h *AdminReportHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserIDFromContext(r.Context())

	schema := &model.ReportSchema{}
	if err := h.readJSON(w, r, &schema); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if err := h.schemas.SaveDraft(r.Context(), schema, user); err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}

	if err := h.writeJSON(w, http.StatusOK, envelope{"schema": schema}, nil); err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}
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
