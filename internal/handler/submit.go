package handler

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"

	"github.com/firewatch/reports/internal/models"
	"github.com/firewatch/reports/internal/security"
)

// Report represents a submitted anonymous report
type Report struct {
	Description     string
	MemberCount     string
	ActivityType    string
	Location        string
	LocationDetails string
	DateTime        string
	Identifiers     string
	Equipment       string
	AdditionalInfo  string
	Attachments     []models.Attachment
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
	if strings.TrimSpace(report.Description) == "" {
		http.Error(w, "Description is required", http.StatusBadRequest)
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
		// Log error internally but don't expose details
		http.Error(w, "Submission failed. Please try again.", http.StatusInternalServerError)
		return
	}

	// Redirect to success page (no tracking parameters)
	http.Redirect(w, r, "/submitted.html", http.StatusSeeOther)
}

// extractReport extracts and sanitizes form fields
func (h *SubmitHandler) extractReport(r *http.Request) Report {
	return Report{
		Description:     sanitizeInput(r.FormValue("description")),
		MemberCount:     sanitizeInput(r.FormValue("member_count")),
		ActivityType:    sanitizeInput(r.FormValue("activity_type")),
		Location:        sanitizeInput(r.FormValue("location")),
		LocationDetails: sanitizeInput(r.FormValue("location_details")),
		DateTime:        sanitizeInput(r.FormValue("date_time")),
		Identifiers:     sanitizeInput(r.FormValue("identifiers")),
		Equipment:       sanitizeInput(r.FormValue("equipment")),
		AdditionalInfo:  sanitizeInput(r.FormValue("additional_info")),
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

// ToEmailContent formats the report for email
func (r *Report) ToEmailContent() string {
	var sb strings.Builder

	sb.WriteString("================================\n")
	sb.WriteString("ANONYMOUS COMMUNITY REPORT\n")
	sb.WriteString("================================\n\n")

	sb.WriteString("INCIDENT DESCRIPTION:\n")
	sb.WriteString(r.Description)
	sb.WriteString("\n\n")

	sb.WriteString("DETAILS:\n")
	sb.WriteString(fmt.Sprintf("- Estimated individuals: %s\n", valueOrDefault(r.MemberCount)))
	sb.WriteString(fmt.Sprintf("- Activity observed: %s\n", valueOrDefault(r.ActivityType)))
	sb.WriteString(fmt.Sprintf("- Location: %s\n", valueOrDefault(r.Location)))
	sb.WriteString(fmt.Sprintf("- Location details: %s\n", valueOrDefault(r.LocationDetails)))
	sb.WriteString(fmt.Sprintf("- Date/Time observed: %s\n", valueOrDefault(r.DateTime)))
	sb.WriteString(fmt.Sprintf("- Identifying features: %s\n", valueOrDefault(r.Identifiers)))
	sb.WriteString(fmt.Sprintf("- Equipment/Vehicles: %s\n", valueOrDefault(r.Equipment)))
	sb.WriteString("\n")

	sb.WriteString("ADDITIONAL INFORMATION:\n")
	sb.WriteString(valueOrDefault(r.AdditionalInfo))
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("ATTACHMENTS: %d file(s)\n", len(r.Attachments)))
	sb.WriteString("================================\n")

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
