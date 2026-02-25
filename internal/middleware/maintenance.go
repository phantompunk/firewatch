package middleware

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/firewatch/internal/model"
)

type maintenanceSettingsLoader interface {
	Load(ctx context.Context) (*model.AppSettings, error)
}

// MaintenanceMode returns a middleware that blocks public routes with a 503
// when maintenance mode is enabled in settings.
func MaintenanceMode(settings maintenanceSettingsLoader, tmpl *template.Template) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s, err := settings.Load(r.Context())
			if err != nil || s.MaintenanceMode {
				if strings.HasPrefix(r.URL.Path, "/api/") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusServiceUnavailable)
					_, _ = w.Write([]byte(`{"error":"service unavailable"}`))
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusServiceUnavailable)
				if execErr := tmpl.ExecuteTemplate(w, "maintenance.html", nil); execErr != nil {
					slog.Error("maintenance: template error", "err", execErr)
				}
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
