package middleware

import (
	"net/http"

	"github.com/firewatch/internal/model"
)

// RequireRole returns middleware that allows only users with the specified role.
// Returns 403 Forbidden for any other role.
func RequireRole(role model.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if RoleFromContext(r.Context()) != role {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireSuperAdmin returns middleware that allows only super_admin users.
// Returns 403 Forbidden for any other role.
func RequireSuperAdmin() func(http.Handler) http.Handler {
	return RequireRole(model.RoleSuperAdmin)
}
