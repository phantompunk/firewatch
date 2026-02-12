package app

import (
	"bytes"
	"embed"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"text/template"

	"github.com/firewatch/reports/internal/models"
)

//go:embed templates/email.tmpl
var emailTemplateFS embed.FS

var emailTmpl = template.Must(template.ParseFS(emailTemplateFS, "templates/email.tmpl"))

// Report represents a submitted anonymous report (SALUTE format)
type Report struct {
	Size           string // S - Size (personnel and vehicles)
	Activity       string // A - Activity (what happened) - Required
	Location       string // L - Location
	Uniform        string // U - Uniform (badges, agency)
	Time           string // T - Time
	Equipment      string // E - Equipment (vehicles, weapons, gear)
	AdditionalInfo string
	Lang           string // Language (en/es)
	Attachments    []models.Attachment
}

func (app *App) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (app *App) submitHandler(w http.ResponseWriter, r *http.Request) {
	app.logger.Info("submission received")

	// Check rate limit (global, not per-IP for privacy)
	if !app.rateLimiter.Allow() {
		app.logger.Warn("submission rate limited")
		http.Error(w, "Please try again later", http.StatusTooManyRequests)
		return
	}

	// Parse multipart form with size limit
	maxSize := int64(app.config.MaxUploadSizeMB) << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	if err := r.ParseMultipartForm(maxSize); err != nil {
		app.logger.Warn("form parse failed", "error", err)
		http.Error(w, "Form too large or invalid", http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()

	// Extract and sanitize form fields
	report := extractReport(r)

	// Validate required fields
	if strings.TrimSpace(report.Activity) == "" {
		app.logger.Warn("submission rejected: missing activity")
		http.Error(w, "Activity description is required", http.StatusBadRequest)
		return
	}

	// Process file attachments
	attachments, err := processAttachments(r)
	if err != nil {
		app.logger.Warn("attachment processing failed", "error", err)
		http.Error(w, "Error processing attachments", http.StatusBadRequest)
		return
	}
	report.Attachments = attachments

	// Log only non-identifying metadata: which fields were filled and attachment count
	app.logger.Info("submission processed",
		"fields_filled", countFilledFields(&report),
		"attachments", len(attachments),
		"lang", report.Lang,
	)

	// Send email
	if err := app.sender.SendReport(report.ToEmailContent(), attachments); err != nil {
		app.logger.Error("email delivery failed", "error", err)
		http.Error(w, "Submission failed. Please try again.", http.StatusInternalServerError)
		return
	}

	app.logger.Info("submission delivered")

	// Redirect to success page (no tracking parameters)
	http.Redirect(w, r, "/submitted.html", http.StatusSeeOther)
}

// countFilledFields returns how many SALUTE fields the reporter filled in (no content logged)
func countFilledFields(r *Report) int {
	count := 0
	for _, v := range []string{r.Size, r.Activity, r.Location, r.Uniform, r.Time, r.Equipment, r.AdditionalInfo} {
		if v != "" {
			count++
		}
	}
	return count
}

// extractReport extracts and sanitizes form fields (SALUTE format)
func extractReport(r *http.Request) Report {
	return Report{
		Size:           sanitizeInput(r.FormValue("size")),
		Activity:       sanitizeInput(r.FormValue("activity")),
		Location:       sanitizeInput(r.FormValue("location")),
		Uniform:        sanitizeInput(r.FormValue("uniform")),
		Time:           sanitizeInput(r.FormValue("time")),
		Equipment:      sanitizeInput(r.FormValue("equipment")),
		AdditionalInfo: sanitizeInput(r.FormValue("additional_info")),
		Lang:           sanitizeInput(r.FormValue("lang")),
	}
}

// processAttachments handles file uploads
func processAttachments(r *http.Request) ([]models.Attachment, error) {
	var attachments []models.Attachment

	files := r.MultipartForm.File["media"]
	if len(files) > 5 {
		return nil, fmt.Errorf("too many files")
	}

	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
		"video/mp4":  true,
		"video/webm": true,
	}

	for _, fileHeader := range files {
		// Check file size (10MB per file)
		if fileHeader.Size > 10<<20 {
			continue
		}

		file, err := fileHeader.Open()
		if err != nil {
			continue
		}

		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			continue
		}

		// Detect content type
		contentType := http.DetectContentType(data)
		if !allowedTypes[contentType] {
			continue
		}

		attachments = append(attachments, models.Attachment{
			Filename:    sanitizeFilename(fileHeader.Filename),
			ContentType: contentType,
			Data:        data,
		})
	}

	return attachments, nil
}

// emailData is the data passed to the email template
type emailData struct {
	Header          string
	Body            string
	AdditionalInfo  string
	AttachmentCount int
}

// ToEmailContent formats the report using the email template
func (r *Report) ToEmailContent() string {
	// Build header: "location, time" / "location" / "time" / ""
	var headerParts []string
	if r.Location != "" {
		headerParts = append(headerParts, r.Location)
	}
	if r.Time != "" {
		headerParts = append(headerParts, r.Time)
	}

	// Build body: "{size} {activity} in {uniform} with {equipment}"
	var bodyParts []string
	if r.Size != "" {
		bodyParts = append(bodyParts, r.Size)
	}
	bodyParts = append(bodyParts, r.Activity)
	if r.Uniform != "" {
		bodyParts = append(bodyParts, "in "+r.Uniform)
	}
	if r.Equipment != "" {
		bodyParts = append(bodyParts, "with "+r.Equipment)
	}

	data := emailData{
		Header:          strings.Join(headerParts, ", "),
		Body:            strings.Join(bodyParts, " "),
		AdditionalInfo:  r.AdditionalInfo,
		AttachmentCount: len(r.Attachments),
	}

	var buf bytes.Buffer
	if err := emailTmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("🚨 SPOTTED\n\n%s\n", r.Activity)
	}
	return buf.String()
}

// sanitizeInput escapes HTML and trims whitespace
func sanitizeInput(s string) string {
	s = strings.TrimSpace(s)
	s = html.EscapeString(s)
	if len(s) > 10000 {
		s = s[:10000]
	}
	return s
}

// sanitizeFilename removes path components and dangerous characters
func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "\x00", "")
	if len(name) > 100 {
		name = name[:100]
	}
	if name == "" {
		name = "attachment"
	}
	return name
}
