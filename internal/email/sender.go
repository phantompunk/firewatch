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

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
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

// EncryptionReady returns nil if PGP encryption is properly configured and the
// public key can be read and parsed. Returns an error describing what is wrong
// otherwise.
func (s *Sender) EncryptionReady() error {
	keyPath := s.resolvedKeyPath()
	if keyPath == "" {
		return fmt.Errorf("no PGP public key found (checked PGP_PUBLIC_KEY_PATH=%q and /run/secrets/pgp_public_key)", s.pgpPublicKeyPath)
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("cannot read PGP public key at %s: %w", keyPath, err)
	}

	_, err = openpgp.ReadArmoredKeyRing(bytes.NewReader(keyData))
	if err != nil {
		return fmt.Errorf("cannot parse PGP public key at %s: %w", keyPath, err)
	}

	return nil
}

// SendReport sends a report via email, encrypting with PGP if configured.
func (s *Sender) SendReport(content string, attachments []models.Attachment) error {
	// If SMTP is not configured, log to stdout (for development)
	if s.smtpHost == "" {
		fmt.Println("=== EMAIL WOULD BE SENT ===")
		fmt.Println(content)
		fmt.Printf("Attachments: %d\n", len(attachments))
		fmt.Println("=== END EMAIL ===")
		return nil
	}

	var msg []byte
	var err error

	keyPath := s.resolvedKeyPath()
	if keyPath != "" {
		msg, err = s.buildEncryptedEmail(content, attachments, keyPath)
	} else {
		msg, err = s.buildEmail(content, attachments)
	}
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

// buildEncryptedEmail builds a PGP/MIME encrypted email (RFC 3156).
// The full MIME body (text + attachments) is encrypted as a single blob.
func (s *Sender) buildEncryptedEmail(content string, attachments []models.Attachment, keyPath string) ([]byte, error) {
	// Build the inner MIME body to encrypt
	innerBody, err := s.buildMIMEBody(content, attachments)
	if err != nil {
		return nil, fmt.Errorf("building MIME body: %w", err)
	}

	encrypted, err := encryptWithPGP(innerBody, keyPath)
	if err != nil {
		return nil, fmt.Errorf("pgp encryption: %w", err)
	}

	// Wrap in PGP/MIME envelope (RFC 3156)
	var buf bytes.Buffer
	envelope := multipart.NewWriter(&buf)

	var emailBuf bytes.Buffer
	emailBuf.WriteString(fmt.Sprintf("From: %s\r\n", s.fromEmail))
	emailBuf.WriteString(fmt.Sprintf("To: %s\r\n", s.recipientEmail))
	emailBuf.WriteString("Subject: Community Report Received\r\n")
	emailBuf.WriteString("MIME-Version: 1.0\r\n")
	emailBuf.WriteString(fmt.Sprintf("Content-Type: multipart/encrypted; protocol=\"application/pgp-encrypted\"; boundary=%s\r\n", envelope.Boundary()))
	emailBuf.WriteString("\r\n")

	// Part 1: PGP/MIME version identification
	versionHeader := textproto.MIMEHeader{}
	versionHeader.Set("Content-Type", "application/pgp-encrypted")
	versionPart, err := envelope.CreatePart(versionHeader)
	if err != nil {
		return nil, err
	}
	versionPart.Write([]byte("Version: 1\r\n"))

	// Part 2: encrypted payload
	encHeader := textproto.MIMEHeader{}
	encHeader.Set("Content-Type", "application/octet-stream")
	encPart, err := envelope.CreatePart(encHeader)
	if err != nil {
		return nil, err
	}
	encPart.Write(encrypted)

	envelope.Close()
	emailBuf.Write(buf.Bytes())
	return emailBuf.Bytes(), nil
}

// buildMIMEBody builds the inner MIME content (text + attachments) without email headers.
func (s *Sender) buildMIMEBody(content string, attachments []models.Attachment) ([]byte, error) {
	if len(attachments) == 0 {
		return []byte(content), nil
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Write a Content-Type header so the decrypted result is parseable
	var body bytes.Buffer
	body.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n\r\n", writer.Boundary()))

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
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			attPart.Write([]byte(encoded[i:end] + "\r\n"))
		}
	}

	writer.Close()
	body.Write(buf.Bytes())
	return body.Bytes(), nil
}

// encryptWithPGP encrypts plaintext using the PGP public key at keyPath.
func encryptWithPGP(plaintext []byte, keyPath string) ([]byte, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading public key: %w", err)
	}

	entityList, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(keyData))
	if err != nil {
		return nil, fmt.Errorf("parsing public key: %w", err)
	}

	var buf bytes.Buffer
	armorWriter, err := armor.Encode(&buf, "PGP MESSAGE", nil)
	if err != nil {
		return nil, fmt.Errorf("creating armor writer: %w", err)
	}

	encWriter, err := openpgp.Encrypt(armorWriter, entityList, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("creating encrypt writer: %w", err)
	}

	if _, err := encWriter.Write(plaintext); err != nil {
		return nil, fmt.Errorf("writing encrypted data: %w", err)
	}
	encWriter.Close()
	armorWriter.Close()

	return buf.Bytes(), nil
}

// resolvedKeyPath returns the absolute path to the PGP key, or empty if not configured.
// It checks the configured path first, then falls back to the Docker secret at
// /run/secrets/pgp_public_key.
func (s *Sender) resolvedKeyPath() string {
	if s.pgpPublicKeyPath != "" {
		path := s.pgpPublicKeyPath
		if !filepath.IsAbs(path) {
			cwd, _ := os.Getwd()
			path = filepath.Join(cwd, path)
		}
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	const dockerSecretPath = "/run/secrets/pgp_public_key"
	if _, err := os.Stat(dockerSecretPath); err == nil {
		return dockerSecretPath
	}

	return ""
}

// sanitizeForEmail ensures content is safe for email
func sanitizeForEmail(s string) string {
	// Remove any potential header injection
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}
