package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "modernc.org/sqlite"

	"github.com/firewatch/internal/auth"
	"github.com/firewatch/internal/config"
	"github.com/firewatch/internal/crypto"
	"github.com/firewatch/internal/db/migrations"
	"github.com/firewatch/internal/mailer"
	"github.com/firewatch/internal/store"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"golang.org/x/sync/errgroup"
	_ "modernc.org/sqlite"
)

type App struct {
	config        *config.Config
	logger        *slog.Logger
	db            *sql.DB
	schemaStore   *store.SchemaStore
	userStore     *store.UserStore
	sessionStore  *store.SessionStore
	settingsStore *store.SettingsStore
	mailer        *mailer.Mailer
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
		return nil, fmt.Errorf("open database: %w", err)
	}

	schemaStore := store.NewSchemaStore(pool)
	sessionStore := store.NewSessionStore(pool)

	encryptKey := make([]byte, 32)
	copy(encryptKey, []byte(cfg.SettingsEncryptionKey)[:32])
	crypter := crypto.New(encryptKey)
	settingsStore := store.NewSettingsStore(pool, crypter)

	hmacKey := make([]byte, 32)
	copy(hmacKey, []byte(cfg.EmailHMACKey)[:32])
	userStore := store.NewUserStore(pool, crypter, hmacKey)

	// TODO: force password reset on first login if seeded from env vars
	auth.SeedFirstAdmin(ctx, userStore)
	if err := schemaStore.SeedDefault(ctx); err != nil {
		slog.Warn("schema seed failed", "err", err)
	}

	s, _ := settingsStore.Load(ctx)
	m := mailer.New(mailer.NewConfigFromSettings(s))

	return &App{
		config:        cfg,
		logger:        logger,
		db:            pool,
		schemaStore:   schemaStore,
		userStore:     userStore,
		sessionStore:  sessionStore,
		settingsStore: settingsStore,
		mailer:        m,
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

func openDB(ctx context.Context, cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("sqlite", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := runMigrations(db); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// One writer at a time — prevents SQLITE_BUSY under concurrent requests
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return db, nil
}

func runMigrations(db *sql.DB) error {
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}

	// 2. Create database driver
	dbDriver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return err
	}

	// 3. Run migrate
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite", dbDriver)
	if err != nil {
		return err
	}

	return m.Up()
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

