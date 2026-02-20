package web

import (
	"embed"
	"html/template"
	"io/fs"
	"log/slog"
)

//go:embed static
var staticFiles embed.FS

//go:embed templates
var templateFiles embed.FS

// StaticFS is the embedded static file system with the "static/" prefix stripped.
var StaticFS fs.FS

// Templates is the compiled template set for all views.
var Templates *template.Template

func init() {
	var err error

	StaticFS, err = fs.Sub(staticFiles, "static")
	if err != nil {
		slog.Error("web: failed to create static FS", "err", err)
		panic(err)
	}

	Templates, err = template.New("").ParseFS(templateFiles,
		"templates/*.html",
		"templates/partials/*.html",
	)
	if err != nil {
		slog.Error("web: failed to parse templates", "err", err)
		panic(err)
	}
}
