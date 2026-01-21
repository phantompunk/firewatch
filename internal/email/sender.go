package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"

	"github.com/firewatch/reports/internal/models"
)

// Sender handles email composition and delivery
type Sender struct {
	smtpHost         string
	smtpPort         int
	smtpUser         string
	smtpPass         string
	fromEmail        string
	recipientEmail   string
	pgpPublicKeyPath string
}

// NewSender creates a new email sender
func NewSender(host string, port int, user, pass, from, recipient, pgpKeyPath string) *Sender {
	return &Sender{
		smtpHost:         host,
		smtpPort:         port,
		smtpUser:         user,
		smtpPass:         pass,
		fromEmail:        from,
		recipientEmail:   recipient,
		pgpPublicKeyPath: pgpKeyPath,
	}
}

// SendReport sends a report via email
func (s *Sender) SendReport(content string, attachments []models.Attachment) error {
	// If SMTP is not configured, log to stdout (for development)
	if s.smtpHost == "" {
		fmt.Println("=== EMAIL WOULD BE SENT ===")
		fmt.Println(content)
		fmt.Printf("Attachments: %d\n", len(attachments))
		fmt.Println("=== END EMAIL ===")
		return nil
	}

	// Build the email
	msg, err := s.buildEmail(content, attachments)
	if err != nil {
		return fmt.Errorf("failed to build email: %w", err)
	}

	// Send via SMTP
	auth := smtp.PlainAuth("", s.smtpUser, s.smtpPass, s.smtpHost)
	addr := fmt.Sprintf("%s:%d", s.smtpHost, s.smtpPort)

	err = smtp.SendMail(addr, auth, s.fromEmail, []string{s.recipientEmail}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// buildEmail constructs the email with attachments
func (s *Sender) buildEmail(content string, attachments []models.Attachment) ([]byte, error) {
	var buf bytes.Buffer

	// If we have attachments, use multipart
	if len(attachments) > 0 {
		return s.buildMultipartEmail(content, attachments)
	}

	// Simple text email
	buf.WriteString(fmt.Sprintf("From: %s\r\n", s.fromEmail))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", s.recipientEmail))
	buf.WriteString("Subject: Community Report Received\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(content)

	return buf.Bytes(), nil
}

// buildMultipartEmail constructs an email with attachments
func (s *Sender) buildMultipartEmail(content string, attachments []models.Attachment) ([]byte, error) {
	var buf bytes.Buffer

	writer := multipart.NewWriter(&buf)

	// Headers
	headers := fmt.Sprintf("From: %s\r\n", s.fromEmail)
	headers += fmt.Sprintf("To: %s\r\n", s.recipientEmail)
	headers += "Subject: Community Report Received\r\n"
	headers += "MIME-Version: 1.0\r\n"
	headers += fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n", writer.Boundary())
	headers += "\r\n"

	var emailBuf bytes.Buffer
	emailBuf.WriteString(headers)

	// Text part
	textHeader := textproto.MIMEHeader{}
	textHeader.Set("Content-Type", "text/plain; charset=utf-8")
	textPart, err := writer.CreatePart(textHeader)
	if err != nil {
		return nil, err
	}
	textPart.Write([]byte(content))

	// Attachment parts
	for _, att := range attachments {
		attHeader := textproto.MIMEHeader{}
		attHeader.Set("Content-Type", att.ContentType)
		attHeader.Set("Content-Transfer-Encoding", "base64")
		attHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", att.Filename))

		attPart, err := writer.CreatePart(attHeader)
		if err != nil {
			return nil, err
		}

		encoded := base64.StdEncoding.EncodeToString(att.Data)
		// Write in 76-character lines per RFC 2045
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			attPart.Write([]byte(encoded[i:end] + "\r\n"))
		}
	}

	writer.Close()

	emailBuf.Write(buf.Bytes())
	return emailBuf.Bytes(), nil
}

// LoadPGPKey loads the organization's public key for encryption
func (s *Sender) LoadPGPKey() ([]byte, error) {
	if s.pgpPublicKeyPath == "" {
		return nil, nil
	}

	// Resolve path
	path := s.pgpPublicKeyPath
	if !filepath.IsAbs(path) {
		cwd, _ := os.Getwd()
		path = filepath.Join(cwd, path)
	}

	return os.ReadFile(path)
}

// sanitizeForEmail ensures content is safe for email
func sanitizeForEmail(s string) string {
	// Remove any potential header injection
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}
