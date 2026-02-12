package app

import (
	"bytes"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/firewatch/reports/internal/config"
	"github.com/firewatch/reports/internal/email"
	"github.com/firewatch/reports/internal/security"
)

func newTestApp() *App {
	return &App{
		config: &config.Config{
			MaxUploadSizeMB: 50,
		},
		logger:      slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
		sender:      email.NewSender("", 0, "", "", "", "", ""),
		rateLimiter: security.NewRateLimiter(100),
	}
}

// validTimestamp returns a timestamp string representing 10 seconds ago
func validTimestamp() string {
	return fmt.Sprintf("%d", time.Now().Unix()-10)
}

// tooFastTimestamp returns a timestamp under the 3-second minimum
func tooFastTimestamp() string {
	return fmt.Sprintf("%d", time.Now().Unix()-1)
}

// tooOldTimestamp returns a timestamp over the 1-hour maximum
func tooOldTimestamp() string {
	return fmt.Sprintf("%d", time.Now().Unix()-3601)
}

// buildMultipartForm creates a multipart form body from key-value pairs
func buildMultipartForm(fields map[string]string) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for k, v := range fields {
		writer.WriteField(k, v)
	}
	writer.Close()
	return body, writer.FormDataContentType()
}

func postSubmit(app *App, fields map[string]string) *httptest.ResponseRecorder {
	body, contentType := buildMultipartForm(fields)
	req := httptest.NewRequest(http.MethodPost, "/api/submit", body)
	req.Header.Set("Content-Type", contentType)
	rr := httptest.NewRecorder()
	app.submitHandler(rr, req)
	return rr
}

func TestHoneypotFilled_RejectsSilently(t *testing.T) {
	rr := postSubmit(newTestApp(), map[string]string{
		"activity": "test activity",
		"website":  "http://spam.com",
		"_t":       validTimestamp(),
	})

	if rr.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/submitted.html" {
		t.Errorf("expected redirect to /submitted.html, got %q", loc)
	}
}

func TestHoneypotEmpty_AllowsSubmission(t *testing.T) {
	rr := postSubmit(newTestApp(), map[string]string{
		"activity": "test activity",
		"website":  "",
		"_t":       validTimestamp(),
	})

	if rr.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/submitted.html" {
		t.Errorf("expected redirect to /submitted.html, got %q", loc)
	}
}

func TestTimestampTooFast_RejectsSilently(t *testing.T) {
	rr := postSubmit(newTestApp(), map[string]string{
		"activity": "test activity",
		"_t":       tooFastTimestamp(),
	})

	if rr.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/submitted.html" {
		t.Errorf("expected redirect to /submitted.html, got %q", loc)
	}
}

func TestTimestampTooOld_RejectsSilently(t *testing.T) {
	rr := postSubmit(newTestApp(), map[string]string{
		"activity": "test activity",
		"_t":       tooOldTimestamp(),
	})

	if rr.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/submitted.html" {
		t.Errorf("expected redirect to /submitted.html, got %q", loc)
	}
}

func TestTimestampMissing_RejectsSilently(t *testing.T) {
	rr := postSubmit(newTestApp(), map[string]string{
		"activity": "test activity",
	})

	if rr.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/submitted.html" {
		t.Errorf("expected redirect to /submitted.html, got %q", loc)
	}
}
