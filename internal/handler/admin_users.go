package handler

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"

	appmw "github.com/firewatch/internal/middleware"
	"github.com/firewatch/internal/mailer"
	"github.com/firewatch/internal/model"
	"github.com/go-chi/chi/v5"
)

type userManagementStore interface {
	ListAll(ctx context.Context) ([]model.AdminUser, error)
	GetByID(ctx context.Context, id string) (*model.AdminUser, error)
	UpdateRoleAndStatus(ctx context.Context, id string, role model.Role, status model.Status) error
	Delete(ctx context.Context, id string) error
}

type allSessionDeleter interface {
	DeleteAllByUserID(ctx context.Context, userID string) error
}

// UsersHandler handles super-admin user management.
type UsersHandler struct {
	users     userManagementStore
	sessions  allSessionDeleter
	mailer    *mailer.Mailer
	templates *template.Template
}

func NewUsersHandler(users userManagementStore, sessions allSessionDeleter, m *mailer.Mailer, tmpl *template.Template) *UsersHandler {
	return &UsersHandler{users: users, sessions: sessions, mailer: m, templates: tmpl}
}

// Page renders the user management page.
func (h *UsersHandler) Page(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.ListAll(r.Context())
	if err != nil {
		slog.Error("users: failed to list", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := h.templates.ExecuteTemplate(w, "admin_users.html", users); err != nil {
		slog.Error("users: template error", "err", err)
	}
}

// List returns all admin users as JSON.
func (h *UsersHandler) List(w http.ResponseWriter, r *http.Request) {
	// TODO: implement
	w.WriteHeader(http.StatusNotImplemented)
}

// Invite sends an invitation to a new admin user.
func (h *UsersHandler) Invite(w http.ResponseWriter, r *http.Request) {
	// TODO: implement (Phase 6)
	w.WriteHeader(http.StatusNotImplemented)
}

// Update changes a user's role or status.
func (h *UsersHandler) Update(w http.ResponseWriter, r *http.Request) {
	// TODO: implement
	_ = chi.URLParam(r, "id")
	_ = appmw.UserIDFromContext(r.Context())
	w.WriteHeader(http.StatusNotImplemented)
}

// Delete removes a user account.
func (h *UsersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	callerID := appmw.UserIDFromContext(r.Context())

	if id == callerID {
		http.Error(w, "Cannot delete your own account", http.StatusBadRequest)
		return
	}

	if err := h.users.Delete(r.Context(), id); err != nil {
		slog.Error("users: failed to delete", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	_ = h.sessions.DeleteAllByUserID(r.Context(), id)
	w.WriteHeader(http.StatusOK)
}
