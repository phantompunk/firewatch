package handler

import (
	"context"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/firewatch/internal/auth"
	appmw "github.com/firewatch/internal/middleware"
	"github.com/firewatch/internal/model"
	"github.com/firewatch/internal/store"
)

type userGetterByEmail interface {
	GetByEmail(ctx context.Context, email string) (*model.AdminUser, string, error)
	UpdateLastLogin(ctx context.Context, id string) error
}

type sessionCreatorDeleter interface {
	Create(ctx context.Context, userID string) (string, error)
	DeleteAllByUserID(ctx context.Context, userID string) error
}

type inviteStore interface {
	GetInviteByToken(ctx context.Context, rawToken string) (*model.Invite, error)
	AcceptInvite(ctx context.Context, inviteID, userID, email, passwordHash, role string) error
}

type acceptInvitePageData struct {
	Token string
	Email string
	Error string
}

// AuthHandler handles admin authentication.
type AuthHandler struct {
	users         userGetterByEmail
	sessions      sessionCreatorDeleter
	invites       inviteStore
	templates     *template.Template
	secureCookies bool
}

func NewAuthHandler(users userGetterByEmail, sessions sessionCreatorDeleter, invites inviteStore, tmpl *template.Template, secureCookies bool) *AuthHandler {
	return &AuthHandler{users: users, sessions: sessions, invites: invites, templates: tmpl, secureCookies: secureCookies}
}

// LoginPage renders the admin login form.
func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "admin_login.html", nil); err != nil {
		slog.Error("auth: template error", "err", err)
	}
}

// Login authenticates an admin and issues a session cookie.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	email := r.FormValue("email")
	password := r.FormValue("password")

	user, hash, err := h.users.GetByEmail(r.Context(), email)
	if err != nil || !auth.Verify(hash, password) {
		if err := h.templates.ExecuteTemplate(w, "admin_login.html", map[string]any{"Error": "Invalid email or password."}); err != nil {
			slog.Error("auth: template error", "err", err)
		}
		return
	}

	if user.Status != model.StatusActive {
		if err := h.templates.ExecuteTemplate(w, "admin_login.html", map[string]any{"Error": "Account is inactive."}); err != nil {
			slog.Error("auth: template error", "err", err)
		}
		return
	}

	sessionID, err := h.sessions.Create(r.Context(), user.ID)
	if err != nil {
		slog.Error("auth: failed to create session", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_ = h.users.UpdateLastLogin(r.Context(), user.ID)

	http.SetCookie(w, &http.Cookie{
		Name:     appmw.SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(4 * time.Hour),
	})
	http.Redirect(w, r, "/admin/report", http.StatusSeeOther)
}

// AcceptInvitePage renders the accept-invite page for the given token.
func (h *AuthHandler) AcceptInvitePage(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	data := acceptInvitePageData{Token: token}

	if token != "" {
		invite, err := h.invites.GetInviteByToken(r.Context(), token)
		if err == nil {
			data.Email = invite.Email
		} else {
			data.Error = "This invitation link is invalid or has expired."
		}
	} else {
		data.Error = "This invitation link is invalid or has expired."
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "accept_invite.html", data); err != nil {
		slog.Error("auth: template error", "err", err)
	}
}

// AcceptInvite handles the form submission for accepting an invitation.
func (h *AuthHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	token := r.FormValue("token")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	renderError := func(email, msg string) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = h.templates.ExecuteTemplate(w, "accept_invite.html", acceptInvitePageData{
			Token: token,
			Email: email,
			Error: msg,
		})
	}

	if password == "" || password != confirmPassword {
		renderError("", "Passwords do not match or are empty.")
		return
	}

	invite, err := h.invites.GetInviteByToken(r.Context(), token)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			renderError("", "This invitation link is invalid or has expired.")
			return
		}
		slog.Error("accept-invite: lookup failed", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	hash, err := auth.Hash(password)
	if err != nil {
		slog.Error("accept-invite: hash failed", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	newUserID := auth.NewID()
	if err := h.invites.AcceptInvite(r.Context(), invite.ID, newUserID, invite.Email, hash, string(invite.Role)); err != nil {
		slog.Error("accept-invite: accept failed", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	sessionID, err := h.sessions.Create(r.Context(), newUserID)
	if err != nil {
		slog.Error("accept-invite: session create failed", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     appmw.SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(60 * time.Minute),
	})
	http.Redirect(w, r, "/admin/report", http.StatusSeeOther)
}

// Logout invalidates all sessions for the authenticated user.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	userID := appmw.UserIDFromContext(r.Context())
	if userID != "" {
		_ = h.sessions.DeleteAllByUserID(r.Context(), userID)
	}
	http.SetCookie(w, &http.Cookie{
		Name:    appmw.SessionCookieName,
		Value:   "",
		Path:    "/",
		MaxAge:  -1,
		Expires: time.Unix(0, 0),
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
