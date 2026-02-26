package mailer

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/smtp"
	"strings"
	"sync"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/firewatch/internal/model"
)

// ReportSender sends form submission emails to assigned address.
type ReportSender interface {
	SendReport(body string) error
	CanEncrypt() error
}

// InviteSender sends invitation emails to new users.
type InviteSender interface {
	SendInvite(to, inviteUrl string) error
}

// PingSender sends test emails to verify mailer configuration.
type PingSender interface {
	Ping() error
	Reconfigure(cfg *Config)
}

type Message struct {
	To          []string
	Subject     string
	Body        string
	IsHTML      bool
	Attachments []Attachments
}

type Attachments struct {
	Name        string
	Data        []byte
	ContentType string
}

type Config struct {
	Host         string
	Port         int
	User         string
	Pass         string
	FromName     string
	FromAddress  string
	To           []string
	PGPPublicKey string
}

type Mailer struct {
	mu     sync.RWMutex
	cfg    *Config
	sendFn func(msg Message) error
}

func New(cfg *Config) *Mailer {
	m := &Mailer{cfg: cfg}
	m.sendFn = m.send
	return m
}

// Reconfigure updates the mailer with new settings.
func (m *Mailer) Reconfigure(cfg *Config) {
	m.mu.Lock()
	m.cfg = cfg
	m.mu.Unlock()
}

// formatMessage constructs the raw email message string from the Message struct.
func (m *Mailer) formatMessage(msg Message) string {
	return fmt.Sprintf(
		"From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		m.cfg.FromName,
		m.cfg.FromAddress,
		strings.Join(msg.To, ", "),
		msg.Subject,
		msg.Body,
	)
}

// send sends an email message over SMTP with mandatory STARTTLS.
func (m *Mailer) send(msg Message) error {
	m.mu.RLock()
	cfg := m.cfg
	m.mu.RUnlock()

	auth := smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); !ok {
		return fmt.Errorf("SMTP server does not support STARTTLS")
	}

	tlsConfig := &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS: %w", err)
	}

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	if err := client.Mail(cfg.FromAddress); err != nil {
		return fmt.Errorf("set from: %w", err)
	}

	for _, recipient := range msg.To {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("set recipient %s: %w", recipient, err)
		}
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("get data writer: %w", err)
	}
	defer wc.Close()

	if _, err := wc.Write([]byte(m.formatMessage(msg))); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

// sendEncrypted encrypts msg.Body with the configured PGP key then sends it.
func (m *Mailer) sendEncrypted(msg Message) error {
	m.mu.RLock()
	key := m.cfg.PGPPublicKey
	m.mu.RUnlock()

	if key == "" {
		return fmt.Errorf("PGP public key is not configured")
	}

	encrypted, err := encryptBody(key, msg.Body)
	if err != nil {
		return fmt.Errorf("encrypt message body: %w", err)
	}

	msg.Body = encrypted
	msg.IsHTML = false

	return m.sendFn(msg)
}

// CanEncrypt validates that the configured PGP public key is non-empty and parseable.
func (m *Mailer) CanEncrypt() error {
	m.mu.RLock()
	key := m.cfg.PGPPublicKey
	m.mu.RUnlock()

	if key == "" {
		return fmt.Errorf("no PGP public key configured")
	}

	keyring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(key))
	if err != nil {
		return fmt.Errorf("invalid PGP public key: %w", err)
	}

	if len(keyring) == 0 {
		return fmt.Errorf("PGP key parsed but no keys found in keyring")
	}

	return nil
}

// encryptBody encrypts plainText for publicKey and returns an ASCII-armored PGP message.
func encryptBody(publicKey, plainText string) (string, error) {
	keyring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(publicKey))
	if err != nil {
		return "", fmt.Errorf("pgp: read recipient key: %w", err)
	}
	if len(keyring) == 0 {
		return "", fmt.Errorf("pgp: no keys found in keyring")
	}

	var buf bytes.Buffer

	armorWriter, err := armor.Encode(&buf, "PGP MESSAGE", nil)
	if err != nil {
		return "", fmt.Errorf("pgp: create armor writer: %w", err)
	}

	plainTextWriter, err := openpgp.Encrypt(armorWriter, keyring, nil, nil, nil)
	if err != nil {
		return "", fmt.Errorf("pgp: encrypt: %w", err)
	}

	if _, err := io.WriteString(plainTextWriter, plainText); err != nil {
		return "", fmt.Errorf("pgp write plaintext: %w", err)
	}

	if err := plainTextWriter.Close(); err != nil {
		return "", fmt.Errorf("pgp: close plaintext writer: %w", err)
	}

	if err := armorWriter.Close(); err != nil {
		return "", fmt.Errorf("pgp: close armor writer: %w", err)
	}

	return buf.String(), nil
}

// Ping connects and authenticates with the SMTP server to verify configuration.
// It requires STARTTLS — consistent with the enforcement in send().
func (m *Mailer) Ping() error {
	m.mu.RLock()
	cfg := m.cfg
	m.mu.RUnlock()

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)

	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("mailer ping: dial %s: %w", addr, err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); !ok {
		return fmt.Errorf("SMTP server does not support STARTTLS")
	}

	tlsConfig := &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("mailer ping: STARTTLS: %w", err)
	}

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("mailer ping: auth: %w", err)
	}

	return nil
}

// SendInvite emails an invitation link directly to the invitee.
func (m *Mailer) SendInvite(toEmail, inviteURL string) error {
	return m.sendFn(Message{
		To:      []string{toEmail},
		Subject: "You've been invited to Firewatch",
		Body: fmt.Sprintf(
			"You have been invited to access Firewatch.\n\nAccept your invitation:\n%s\n\nThis link expires in 48 hours.",
			inviteURL,
		),
		IsHTML: false,
	})
}

// SendReport encrypts body with PGP and sends it to the configured destination(s).
func (m *Mailer) SendReport(body string) error {
	m.mu.RLock()
	to := m.cfg.To
	m.mu.RUnlock()

	return m.sendEncrypted(Message{
		To:      to,
		Subject: "Report from Firewatch",
		Body:    body,
		IsHTML:  false,
	})
}

// NewConfigFromSettings creates a mailer Config from application settings.
func NewConfigFromSettings(s *model.AppSettings) *Config {
	return &Config{
		Host:         s.SMTPHost,
		Port:         s.SMTPPort,
		User:         s.SMTPUser,
		Pass:         s.SMTPPass,
		FromName:     s.SMTPFromName,
		FromAddress:  s.SMTPFromAddress,
		To:           []string{s.DestinationEmail},
		PGPPublicKey: s.PGPKey,
	}
}
