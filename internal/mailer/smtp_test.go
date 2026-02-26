package mailer

import (
	"io"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
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

func generateTestKey(t *testing.T) (publickey, privatekey string) {
	t.Helper()

	entity, err := openpgp.NewEntity("Test User", "", "test@example.org", nil)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}

	var pubBuf, privBuf strings.Builder
	pubWriter, _ := armor.Encode(&pubBuf, "PGP PUBLIC KEY BLOCK", nil)
	entity.Serialize(pubWriter)
	pubWriter.Close()

	privWriter, _ := armor.Encode(&privBuf, "PGP PRIVATE KEY BLOCK", nil)
	entity.SerializePrivate(privWriter, nil)
	privWriter.Close()

	return pubBuf.String(), privBuf.String()
}

func mustDecrypt(t *testing.T, armoredPrivKey, armoredMsg string) string {
	t.Helper()

	keyring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(armoredPrivKey))
	if err != nil {
		t.Fatalf("mustDecrypt: read private key: %v", err)
	}

	block, err := armor.Decode(strings.NewReader(armoredMsg))
	if err != nil {
		t.Fatalf("mustDecrypt: decode armor: %v", err)
	}

	md, err := openpgp.ReadMessage(block.Body, keyring, nil, nil)
	if err != nil {
		t.Fatalf("mustDecrypt: read message: %v", err)
	}

	var buf strings.Builder
	if _, err := io.Copy(&buf, md.UnverifiedBody); err != nil {
		t.Fatalf("mustDecrypt: read body: %v", err)
	}

	return buf.String()
}

func TestSendEncryptedReport(t *testing.T) {
	pubKey, privKey := generateTestKey(t)
	m := New(&Config{
		FromAddress:  "noreply@example.org",
		FromName:     "Firewatch",
		To:           []string{"admin@example.org"},
		PGPPublicKey: pubKey,
	})

	captured := captureSend(t, m)

	if err := m.SendReport("Sensitive info"); err != nil {
		t.Fatalf("send report error: %v", err)
	}

	if !strings.Contains(captured.To[0], "admin@example.org") {
		t.Errorf("unexpected recipient, got %s", captured.To[0])
	}

	if !strings.Contains(captured.Body, "-----BEGIN PGP MESSAGE-----") {
		t.Errorf("expected PGP encrypted body")
	}

	decrypted := mustDecrypt(t, privKey, captured.Body)
	if !strings.Contains(decrypted, "Sensitive info") {
		t.Errorf("decrypted body missing original content, got: %s", decrypted)
	}
}

func TestCanEncryptValidKey(t *testing.T) {
	pubKey, _ := generateTestKey(t)
	m := New(&Config{PGPPublicKey: pubKey})

	if err := m.CanEncrypt(); err != nil {
		t.Errorf("expected nil for valid key, got: %v", err)
	}
}

func TestCanEncryptNoKey(t *testing.T) {
	m := New(&Config{})

	err := m.CanEncrypt()
	if err == nil {
		t.Errorf("expected error for missing key, got: %v", err)
	}
	if !strings.Contains(err.Error(), "no PGP public key configured") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCanEncryptAfterReconfigure(t *testing.T) {
	m := New(&Config{})

	err := m.CanEncrypt()
	if err == nil {
		t.Errorf("expected error before key is configured")
	}

	pubKey, _ := generateTestKey(t)
	m.Reconfigure(&Config{PGPPublicKey: pubKey})

	if err := m.CanEncrypt(); err != nil {
		t.Errorf("expected nil after valid key reconfigured, got: %v", err)
	}
}
