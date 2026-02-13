package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/firewatch/reports/internal/config"
	"github.com/firewatch/reports/internal/email"
	"github.com/firewatch/reports/internal/security"
)

type App struct {
	config *config.Config
	logger *slog.Logger
	sender *email.Sender
	rateLimiter *security.RateLimiter
}

func (app *App) Close() {
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	logger := newLogger(cfg)

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

	// In production, require PGP encryption to be properly configured
	if cfg.IsProduction() {
		if err := emailSender.EncryptionReady(); err != nil {
			return nil, fmt.Errorf("pgp encryption required in production: %w", err)
		}
		logger.Info("pgp encryption verified")
	} else if err := emailSender.EncryptionReady(); err != nil {
		logger.Warn("pgp encryption not available, emails will be sent unencrypted", "reason", err.Error())
	}

	return &App{
		config:      cfg,
		logger:      logger,
		sender:      emailSender,
		rateLimiter:  rateLimiter,
	}, nil
}

func (app App) Start() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", app.config.Port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	shutdownErr := make(chan error)
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		app.logger.Info("shutting down server", "signal", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		shutdownErr <- srv.Shutdown(ctx)
	}()

	app.logger.Info("starting server", "addr", srv.Addr, "env", app.config.Env)

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownErr
	if err != nil {
		return err
	}

	app.logger.Info("stopped server", "addr", srv.Addr)
	return nil
}

func newLogger(cfg *config.Config) *slog.Logger {
	logLevel := slog.LevelInfo

	if cfg.IsDevelopment() {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	slog.SetDefault(logger)
	return logger
}
