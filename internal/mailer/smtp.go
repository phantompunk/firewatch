package mailer

import (
	"fmt"
	"net/smtp"
	"strings"
	"sync"

	"github.com/firewatch/internal/model"
)

type ReportSender interface {
	SendReport(body string) error
}

type InviteSender interface {
	SendInvite(to, inviteUrl string) error
}

// TestSender sends test emails to verify mailer configuration.
type TestSender interface {
	SendTest() error
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
	Host        string
	Port        int
	User        string
	Pass        string
	FromName    string
	FromAddress string
	To          []string
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
	m.cfg = cfg
}

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

func (m *Mailer) send(msg Message) error {
	m.mu.Lock()
	cfg := m.cfg
	m.mu.Unlock()

	// Send via SMTP
	auth := smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	return smtp.SendMail(addr, auth, cfg.FromAddress, msg.To, []byte(m.formatMessage(msg)))
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

// SendTest sends a test email to verify mailer configuration.
func (m *Mailer) SendTest() error {
	return m.sendFn(Message{
		To:      []string{m.cfg.FromAddress},
		Subject: "Test Email from Firewatch",
		Body:    "This is a test email from Firewatch. If you received this, your mailer is configured correctly.",
		IsHTML:  false,
	})
}

// SendReport sends a report email to the configured destination(s).
func (m *Mailer) SendReport(body string) error {
	return m.sendFn(Message{
		To:      m.cfg.To,
		Subject: "Report from Firewatch",
		Body:    body,
		IsHTML:  false,
	})
}

// NewConfigFromSettings creates a mailer Config from application settings.
func NewConfigFromSettings(s *model.AppSettings) *Config {
	return &Config{
		Host:        s.SMTPHost,
		Port:        s.SMTPPort,
		User:        s.SMTPUser,
		Pass:        s.SMTPPass,
		FromName:    s.SMTPFromName,
		FromAddress: s.DestinationEmail,
		To:          []string{s.DestinationEmail},
	}
}
