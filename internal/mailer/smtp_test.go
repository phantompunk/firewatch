package mailer

import (
	"strings"
	"testing"
)

func TestFormatMessageWithPlainText(t *testing.T) {
	cfg := &Config{
		FromName:    "Firewatch",
		FromAddress: "noreply@example.org",
	}

	msg := Message{
		To:      []string{"user@example.org"},
		Subject: "Test Subject",
		Body:    "This is a test email.",
		IsHTML:  false,
	}

	mailer := New(cfg)
	result := mailer.formatMessage(msg)

	cases := []struct {
		name string
		want string
	}{
		{"from header", "From: Firewatch <noreply@example.org>"},
		{"to header", "To: user@example.org"},
		{"subject header", "Subject: Test Subject"},
		{"mime header", "MIME-Version: 1.0"},
		{"content type header", "Content-Type: text/plain; charset=UTF-8"},
		{"body", "\r\nThis is a test email."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(result, tc.want) {
				t.Errorf("expected %q in message, got:\n%s", tc.want, result)
			}
		})
	}
}

func TestFormatMessageWithMultipleRecipients(t *testing.T) {
	cfg := &Config{FromName: "Firewatch", FromAddress: "noreply@example.org"}
	msg := Message{
		To: []string{"a@example.org", "b@example.org"},
	}

	result := New(cfg).formatMessage(msg)
	if !strings.Contains(result, "To: a@example.org, b@example.org") {
		t.Errorf("expected multiple recipients in To header, got:\n%s", result)
	}
}

func captureSend(t *testing.T, m *Mailer) *Message {
	t.Helper()
	var captured Message
	m.sendFn = func(msg Message) error {
		captured = msg
		return nil
	}
	return &captured
}

func TestSendInviteEmail(t *testing.T) {
	m := New(&Config{FromAddress: "noreply@example.org", FromName: "Firewatch"})
	captured := captureSend(t, m)

	inviteURL := "https://example.org/accept-invite?token=abc123"
	if err := m.SendInvite("user@example.org", inviteURL); err != nil {
		t.Fatalf("SendInvite returned an error: %v", err)
	}

	if !strings.Contains(captured.Body, inviteURL) {
		t.Errorf("expected invite URL in body, got %s", captured.Body)
	}

	if !strings.Contains(captured.Subject, "You've been invited to Firewatch") {
		t.Errorf("unexpected subject: %s", captured.Subject)
	}

	if !strings.Contains(captured.Body, "48 hours") {
		t.Errorf("expected expiry in body, got: %s", captured.Body)
	}
}

func TestSendReportEmail(t *testing.T) {
	m := New(&Config{FromAddress: "noreply@example.org", FromName: "Firewatch", To: []string{"admin@example.org"}})
	captured := captureSend(t, m)

	if err := m.SendReport("This is a report."); err != nil {
		t.Fatalf("SendReport returned an error: %v", err)
	}

	// To address should be the configured To address
	if !strings.Contains(captured.To[0], "admin@example.org") {
		t.Errorf("unexpected recipient, got %s", captured.To[0])
	}

	if !strings.Contains(captured.Body, "This is a report.") {
		t.Errorf("unexpected body, got: %s", captured.Body)
	}
}
