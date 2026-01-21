package main

import (
	"log"
	"net/http"
	"os"

	"github.com/firewatch/reports/config"
	"github.com/firewatch/reports/internal/email"
	"github.com/firewatch/reports/internal/handler"
	"github.com/firewatch/reports/internal/security"
)

func main() {
	// Load .env file if present
	config.LoadEnv()

	// Load configuration
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Initialize email sender
	emailSender := email.NewSender(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUser,
		cfg.SMTPPass,
		cfg.FromEmail,
		cfg.RecipientEmail,
		cfg.PGPPublicKeyPath,
	)

	// Initialize rate limiter
	rateLimiter := security.NewRateLimiter(cfg.RateLimitPerMinute)

	// Initialize handlers
	submitHandler := handler.NewSubmitHandler(emailSender, rateLimiter, cfg.MaxUploadSizeMB)

	// Setup routes
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/submit", submitHandler.Handle)
	mux.HandleFunc("/health", handler.HealthHandler)

	// Static files
	staticFS := http.FileServer(http.Dir(cfg.StaticDir))
	mux.Handle("/", staticFS)

	// Wrap with security headers middleware
	secureHandler := security.HeadersMiddleware(mux)

	// Start server
	addr := ":" + cfg.Port
	log.Printf("Server starting on %s", addr)
	log.Printf("Static files served from: %s", cfg.StaticDir)

	// Disable default logging for privacy - only log startup
	server := &http.Server{
		Addr:     addr,
		Handler:  secureHandler,
		ErrorLog: log.New(os.Stderr, "", 0), // Minimal error logging
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
