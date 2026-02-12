package app

import (
	"net/http"

	"github.com/firewatch/reports/internal/security"
	"github.com/julienschmidt/httprouter"
)

func (app *App) routes() http.Handler {
	router := httprouter.New()

	// Static files
	router.NotFound = http.FileServer(http.Dir(app.config.StaticDir))

	// Health and status routes
	router.HandlerFunc(http.MethodGet, "/api/health", app.healthCheckHandler)

	// Public API routes
	router.HandlerFunc(http.MethodPost, "/api/submit", app.submitHandler)

	// Wrap with security headers middleware
	// secureHandler := security.HeadersMiddleware(mux)

	return security.HeadersMiddleware(router)
	// return app.enableCORS(router)
}
