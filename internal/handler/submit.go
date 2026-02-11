package handler

import (
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/firewatch/reports/internal/models"
	"github.com/firewatch/reports/internal/security"
)

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

// SubmitHandler handles anonymous report submissions
type SubmitHandler struct {
	emailSender     models.EmailSender
	rateLimiter     *security.RateLimiter
	maxUploadSizeMB int
}

// NewSubmitHandler creates a new submission handler
func NewSubmitHandler(emailSender models.EmailSender, rateLimiter *security.RateLimiter, maxUploadSizeMB int) *SubmitHandler {
	return &SubmitHandler{
		emailSender:     emailSender,
		rateLimiter:     rateLimiter,
		maxUploadSizeMB: maxUploadSizeMB,
	}
}

// Handle processes form submissions
func (h *SubmitHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Only accept POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check rate limit (global, not per-IP for privacy)
	if !h.rateLimiter.Allow() {
		http.Error(w, "Please try again later", http.StatusTooManyRequests)
		return
	}

	// Parse multipart form with size limit
	maxSize := int64(h.maxUploadSizeMB) << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	if err := r.ParseMultipartForm(maxSize); err != nil {
		http.Error(w, "Form too large or invalid", http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()

	// Extract and sanitize form fields
	report := h.extractReport(r)

	// Validate required fields
	if strings.TrimSpace(report.Activity) == "" {
		http.Error(w, "Activity description is required", http.StatusBadRequest)
		return
	}

	// Process file attachments
	attachments, err := h.processAttachments(r)
	if err != nil {
		http.Error(w, "Error processing attachments", http.StatusBadRequest)
		return
	}
	report.Attachments = attachments

	// Send email
	if err := h.emailSender.SendReport(report.ToEmailContent(), attachments); err != nil {
		// TEMP: Log error for debugging
		log.Printf("ERROR sending email: %v", err)
		http.Error(w, "Submission failed. Please try again.", http.StatusInternalServerError)
		return
	}

	// Redirect to success page (no tracking parameters)
	http.Redirect(w, r, "/submitted.html", http.StatusSeeOther)
}

// extractReport extracts and sanitizes form fields (SALUTE format)
func (h *SubmitHandler) extractReport(r *http.Request) Report {
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
func (h *SubmitHandler) processAttachments(r *http.Request) ([]models.Attachment, error) {
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
			continue // Skip files that are too large
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
			continue // Skip unsupported file types
		}

		attachments = append(attachments, models.Attachment{
			Filename:    sanitizeFilename(fileHeader.Filename),
			ContentType: contentType,
			Data:        data,
		})
	}

	return attachments, nil
}

// ToEmailContent formats the report for email (SALUTE format)
func (r *Report) ToEmailContent() string {
	var sb strings.Builder

	sb.WriteString("=====================================\n")
	sb.WriteString("ANONYMOUS SALUTE REPORT\n")
	if r.Lang == "es" {
		sb.WriteString("(Formato ACTUAR)\n")
	}
	sb.WriteString("=====================================\n\n")

	sb.WriteString("[S] SIZE:\n")
	sb.WriteString(fmt.Sprintf("    %s\n\n", valueOrDefault(r.Size)))

	sb.WriteString("[A] ACTIVITY:\n")
	sb.WriteString(fmt.Sprintf("    %s\n\n", r.Activity))

	sb.WriteString("[L] LOCATION:\n")
	sb.WriteString(fmt.Sprintf("    %s\n\n", valueOrDefault(r.Location)))

	sb.WriteString("[U] UNIFORM:\n")
	sb.WriteString(fmt.Sprintf("    %s\n\n", valueOrDefault(r.Uniform)))

	sb.WriteString("[T] TIME:\n")
	sb.WriteString(fmt.Sprintf("    %s\n\n", valueOrDefault(r.Time)))

	sb.WriteString("[E] EQUIPMENT:\n")
	sb.WriteString(fmt.Sprintf("    %s\n\n", valueOrDefault(r.Equipment)))

	sb.WriteString("ADDITIONAL INFORMATION:\n")
	sb.WriteString(fmt.Sprintf("    %s\n\n", valueOrDefault(r.AdditionalInfo)))

	sb.WriteString(fmt.Sprintf("ATTACHMENTS: %d file(s)\n", len(r.Attachments)))
	sb.WriteString("=====================================\n")

	return sb.String()
}

// sanitizeInput escapes HTML and trims whitespace
func sanitizeInput(s string) string {
	s = strings.TrimSpace(s)
	s = html.EscapeString(s)
	// Limit length to prevent abuse
	if len(s) > 10000 {
		s = s[:10000]
	}
	return s
}

// sanitizeFilename removes path components and dangerous characters
func sanitizeFilename(name string) string {
	// Remove path separators
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	// Remove null bytes
	name = strings.ReplaceAll(name, "\x00", "")
	// Limit length
	if len(name) > 100 {
		name = name[:100]
	}
	if name == "" {
		name = "attachment"
	}
	return name
}

// valueOrDefault returns the value or "Not provided"
func valueOrDefault(s string) string {
	if s == "" {
		return "Not provided"
	}
	return s
}
