package mailer

import (
	"fmt"
	"net/smtp"

	"github.com/firewatch/internal/model"
)

// Mailer sends emails via SMTP.
type Mailer struct {
	settings *model.AppSettings
}

// New returns a Mailer. Call Reconfigure to load settings before sending.
func New() *Mailer {
	return &Mailer{}
}

// Reconfigure updates the mailer with new settings.
func (m *Mailer) Reconfigure(settings *model.AppSettings) {
	m.settings = settings
}

// SendInvite emails an invitation link directly to the invitee.
func (m *Mailer) SendInvite(toEmail, inviteURL string) error {
	if m.settings == nil {
		return fmt.Errorf("mailer: not configured")
	}
	s := m.settings
	subject := "You've been invited to Firewatch"
	body := fmt.Sprintf(
		"You have been invited to access Firewatch.\n\nAccept your invitation:\n%s\n\nThis link expires in 48 hours.",
		inviteURL,
	)
	addr := fmt.Sprintf("%s:%d", s.SMTPHost, s.SMTPPort)
	auth := smtp.PlainAuth("", s.SMTPUser, s.SMTPPass, s.SMTPHost)
	from := fmt.Sprintf("%s <%s>", s.SMTPFromName, s.SMTPFromAddress)
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		from, toEmail, subject, body,
	)
	return smtp.SendMail(addr, auth, s.SMTPFromAddress, []string{toEmail}, []byte(msg))
}

// Send sends an email with the given subject and body to the configured destination.
func (m *Mailer) Send(subject, body string) error {
	if m.settings == nil {
		return fmt.Errorf("mailer: not configured")
	}
	s := m.settings
	addr := fmt.Sprintf("%s:%d", s.SMTPHost, s.SMTPPort)
	auth := smtp.PlainAuth("", s.SMTPUser, s.SMTPPass, s.SMTPHost)

	from := fmt.Sprintf("%s <%s>", s.SMTPFromName, s.SMTPFromAddress)
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		from, s.DestinationEmail, subject, body,
	)

	return smtp.SendMail(addr, auth, s.SMTPFromAddress, []string{s.DestinationEmail}, []byte(msg))
}
