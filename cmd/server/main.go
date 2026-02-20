package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/firewatch/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := app.New()
	if err != nil {
		slog.Error("failed to initialize application", "error", err)
		os.Exit(1)
	}
	defer app.Close()

	if err := app.Start(ctx); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}

	// m := mailer.New()
	// if settings != nil {
	// 	m.Reconfigure(settings)
	// }
}
