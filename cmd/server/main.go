package main

import (
	"log/slog"
	"os"

	"github.com/firewatch/reports/internal/app"
)

func main() {
	app, err := app.New()
	if err != nil {
		slog.Error("failed to initialize application", "error", err)
		os.Exit(1)
	}
	defer app.Close()

	if err := app.Start(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
