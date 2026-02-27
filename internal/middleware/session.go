package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/firewatch/internal/model"
)

const SessionCookieName = "session"

type contextKey string

const (
	contextKeyUserID contextKey = "userID"
	contextKeyRole   contextKey = "role"
)

// SessionReader retrieves the user ID for a session token.
type SessionReader interface {
	GetUserID(ctx context.Context, sessionID string) (string, error)
}

// userByIDer retrieves an admin user by ID.
type userByIDer interface {
	GetByID(ctx context.Context, id string) (*model.AdminUser, error)
}

// SignCookie returns "<sessionID>.<HMAC-SHA256-hex>" signed with key.
// This is the value stored in the session cookie.
func SignCookie(key []byte, sessionID string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(sessionID))
	return sessionID + "." + hex.EncodeToString(mac.Sum(nil))
}

// verifyAndExtract validates the signed cookie value and returns the bare
// session ID. Returns ("", false) if the signature is missing or invalid.
func verifyAndExtract(key []byte, cookieValue string) (string, bool) {
	dot := strings.LastIndex(cookieValue, ".")
	if dot < 0 {
		return "", false
	}
	sessionID := cookieValue[:dot]
	sig := cookieValue[dot+1:]

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(sessionID))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", false
	}
	return sessionID, true
}

// Session middleware validates the session cookie and populates the request
// context with the user ID and role. Unauthenticated requests are redirected
// to /admin/login.
func Session(key []byte, sessions SessionReader, users userByIDer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil {
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
				return
			}

			sessionID, ok := verifyAndExtract(key, cookie.Value)
			if !ok {
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
				return
			}

			userID, err := sessions.GetUserID(r.Context(), sessionID)
			if err != nil {
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
				return
			}

			user, err := users.GetByID(r.Context(), userID)
			if err != nil {
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyUserID, userID)
			ctx = context.WithValue(ctx, contextKeyRole, user.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext returns the authenticated user's ID from the context.
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyUserID).(string)
	return v
}

// RoleFromContext returns the authenticated user's role from the context.
func RoleFromContext(ctx context.Context) model.Role {
	v, _ := ctx.Value(contextKeyRole).(model.Role)
	return v
}

// IsSuperAdmin reports whether the authenticated user has the super_admin role.
func IsSuperAdmin(ctx context.Context) bool {
	return RoleFromContext(ctx) == model.RoleSuperAdmin
}
