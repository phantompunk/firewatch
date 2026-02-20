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

	"github.com/firewatch/internal/auth"
	"github.com/firewatch/internal/config"
	"github.com/firewatch/internal/crypto"
	"github.com/firewatch/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

type App struct {
	config       *config.Config
	logger       *slog.Logger
	db           *pgxpool.Pool
	schemaStore  *store.SchemaStore
	userStore    *store.UserStore
	sessionStore *store.SessionStore
	settingsStore *store.SettingsStore
}

func (app *App) Close() {
	app.db.Close()
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	logger := newLogger(cfg)

	ctx := context.Background()
	pool, err := openDB(ctx, cfg)
	if err != nil {
		logger.Error(err.Error())
	}

	schemaStore := store.NewSchemaStore(pool)
	userStore := store.NewUserStore(pool)
	sessionStore := store.NewSessionStore(pool)

	encryptKey := make([]byte, 32)
	copy(encryptKey, []byte(cfg.SettingsEncryptionKey)[:32])
	crypter := crypto.New(encryptKey)
	settingsStore := store.NewSettingsStore(pool, crypter)

	auth.SeedFirstAdmin(ctx, userStore)
	if err := schemaStore.SeedDefault(ctx); err != nil {
		slog.Warn("schema seed failed", "err", err)
	}

	return &App{
		config:       cfg,
		logger:       logger,
		db:           pool,
		schemaStore:  schemaStore,
		userStore:    userStore,
		sessionStore: sessionStore,
		settingsStore: settingsStore,
	}, nil
}

func (app App) Start(ctx context.Context) error {
	// Create an errgroup derived from the parent context
	g, gctx := errgroup.WithContext(ctx)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", app.config.Port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	// Start the server in a goroutine
	g.Go(func() error {
		app.logger.Info("starting server", "addr", srv.Addr, "env", app.config.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.logger.Error("server failed", "error", err)
		}
		return nil
	})

	// Start shutdown listener
	g.Go(func() error {
		<-gctx.Done() // Wait for OS signal or parent context to fail

		app.logger.Info("shutting down server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	app.logger.Info("stopped server")
	return nil
}

func openDB(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	// Verify database connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return pool, nil
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

func mustEnv(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	slog.Error("missing required environment variable", "key", key)
	os.Exit(1)
	return ""
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
