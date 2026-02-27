package app

import (
	"net/http"
	"time"

	"github.com/firewatch/internal/handler"
	"github.com/firewatch/internal/middleware"
	"github.com/firewatch/internal/web"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"golang.org/x/time/rate"
)

func (app App) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(middleware.SecurityHeaders)

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServerFS(web.StaticFS)))

	// Health check
	r.Get("/api/health", handler.Health(app.db))

	// Public report form
	reportHandler := handler.NewReportHandler(app.logger, app.schemaStore, app.sessionStore, app.mailerQueue, web.Templates)
	r.Get("/admin", reportHandler.RedirectToLogin)
	r.Get("/login", reportHandler.RedirectToLogin)

	// Maintenance-guarded public routes
	maintenanceMW := middleware.MaintenanceMode(app.settingsStore, web.Templates)
	ratelimitMW := middleware.RateLimit(rate.Every(time.Minute/10), 5) // 10 requests per minute with burst of 5
	r.Group(func(r chi.Router) {
		r.Use(maintenanceMW)
		r.Get("/", reportHandler.Form)
		r.Get("/api/report", reportHandler.Get)
		r.With(ratelimitMW).Post("/api/report", reportHandler.Submit)
	})

	// Admin auth (public endpoints)
	loginRatelimitMW := middleware.RateLimit(rate.Every(10*time.Minute/5), 5) // 5 login attempts per 10 minutes with burst of 5
	authHandler := handler.NewAuthHandler(app.userStore, app.sessionStore, app.userStore, web.Templates, app.config.SecureCookies, app.config.SessionSecret)
	r.Get("/admin/login", authHandler.LoginPage)
	r.With(loginRatelimitMW).Post("/api/admin/login", authHandler.Login)
	r.Get("/accept-invite", authHandler.AcceptInvitePage)
	r.Post("/api/accept-invite", authHandler.AcceptInvite)

	// Protected admin routes
	sessionMW := middleware.Session(app.config.SessionSecret, app.sessionStore, app.userStore)
	r.Group(func(r chi.Router) {
		r.Use(sessionMW)

		r.Post("/api/admin/logout", authHandler.Logout)

		adminReportHandler := handler.NewAdminReportHandler(app.logger, app.schemaStore, web.Templates)
		r.Get("/admin/report", adminReportHandler.Page)
		r.Get("/api/admin/report", adminReportHandler.Get)
		r.Put("/api/admin/report", adminReportHandler.Update)
		r.Post("/api/admin/report/apply", adminReportHandler.Apply)
		r.Post("/api/admin/report/revert", adminReportHandler.Revert)

		settingsHandler := handler.NewSettingsHandler(app.logger, app.settingsStore, app.mailerQueue, web.Templates)
		r.Get("/admin/settings", settingsHandler.Page)
		r.Get("/api/admin/settings", settingsHandler.Get)
		r.Put("/api/admin/settings", settingsHandler.Update)
		r.Post("/api/admin/settings/apply", settingsHandler.Apply)
		r.Post("/api/admin/settings/test-email", settingsHandler.TestEmail)

		// Super admin only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireSuperAdmin())

			usersHandler := handler.NewUsersHandler(app.userStore, app.sessionStore, app.mailerQueue, app.config.AdminInviteBaseURL, web.Templates)
			r.Get("/admin/users", usersHandler.Page)
			r.Get("/api/admin/users", usersHandler.List)
			r.Post("/api/admin/users", usersHandler.Invite)
			r.Put("/api/admin/users/{id}", usersHandler.Update)
			r.Delete("/api/admin/users/{id}", usersHandler.Delete)
		})
	})
	return r
}
