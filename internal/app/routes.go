package app

import (
	"net/http"

	"github.com/firewatch/internal/handler"
	"github.com/firewatch/internal/middleware"
	"github.com/firewatch/internal/web"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func (app App) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServerFS(web.StaticFS)))

	// Health check
	r.Get("/api/health", handler.Health(app.db))

	// Public report form
	reportHandler := handler.NewReportHandler(app.logger, app.schemaStore, app.mailer, web.Templates)
	r.Get("/", reportHandler.Form)
	r.Get("/api/report", reportHandler.Get)

	// TODO: finish
	r.Post("/api/report", reportHandler.Submit)

	// Admin auth (public endpoints)
	authHandler := handler.NewAuthHandler(app.userStore, app.sessionStore, web.Templates, app.config.SecureCookies)
	r.Get("/admin/login", authHandler.LoginPage)
	r.Post("/api/admin/login", authHandler.Login)

	// Protected admin routes
	sessionMW := middleware.Session(app.sessionStore, app.userStore)
	r.Group(func(r chi.Router) {
		r.Use(sessionMW)

		r.Post("/api/admin/logout", authHandler.Logout)

		adminReportHandler := handler.NewAdminReportHandler(app.logger, app.schemaStore, web.Templates)
		r.Get("/admin/report", adminReportHandler.Page)
		r.Get("/api/admin/report", adminReportHandler.Get)
		r.Put("/api/admin/report", adminReportHandler.Update)
		// TODO: finish
		r.Post("/api/admin/report/apply", adminReportHandler.Apply)

		settingsHandler := handler.NewSettingsHandler(app.logger, app.settingsStore, app.mailer, web.Templates)
		r.Get("/admin/settings", settingsHandler.Page)
		r.Get("/api/admin/settings", settingsHandler.Get)
		r.Put("/api/admin/settings", settingsHandler.Update)
		r.Post("/api/admin/settings/apply", settingsHandler.Apply)
		r.Post("/api/admin/settings/test-email", settingsHandler.TestEmail)

		// Super admin only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("super_admin"))

			usersHandler := handler.NewUsersHandler(app.userStore, app.sessionStore, nil, web.Templates)
			r.Get("/admin/users", usersHandler.Page)
			r.Get("/api/admin/users", usersHandler.List)
			r.Post("/api/admin/users", usersHandler.Invite)
			r.Put("/api/admin/users/{id}", usersHandler.Update)
			r.Delete("/api/admin/users/{id}", usersHandler.Delete)
		})
	})
	return r
}
