package models

// Attachment represents an uploaded file
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// EmailSender defines the interface for sending reports
type EmailSender interface {
	SendReport(content string, attachments []Attachment) error
}
