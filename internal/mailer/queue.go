package mailer

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type queuedMessage struct {
	msg     Message
	retries int
}

type Queue struct {
	mailer   *Mailer
	ch       chan queuedMessage
	rate     time.Duration
	maxRetry int
}

func NewQueue(m *Mailer, rate time.Duration, bufferSize, maxRetry int) *Queue {
	return &Queue{
		mailer:   m,
		ch:       make(chan queuedMessage, bufferSize),
		rate:     rate,
		maxRetry: maxRetry,
	}
}

// Start processes queued messages at the configured rate until ctx is cancelled.
// On shutdown it drains any remaining messages before returning.
func (q *Queue) Start(ctx context.Context) {
	ticker := time.NewTicker(q.rate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			q.drain()
			return
		case <-ticker.C:
			select {
			case item := <-q.ch:
				q.attempt(ctx, item)
			default:
				// no message ready; wait for next tick
			}
		}
	}
}

// Enqueue adds a pre-encrypted message to the queue. Messages must already
// have their body encrypted before enqueuing — see QueuedMailer.
func (q *Queue) Enqueue(msg Message) error {
	select {
	case q.ch <- queuedMessage{msg: msg}:
		return nil
	default:
		return fmt.Errorf("mailer: queue full, message not queued")
	}
}

// attempt sends a message, scheduling a context-aware retry with backoff on failure.
func (q *Queue) attempt(ctx context.Context, item queuedMessage) {
	if err := q.mailer.send(item.msg); err == nil {
		return
	}

	if item.retries >= q.maxRetry {
		slog.Error("mailer: message dropped after max retries", "to", item.msg.To, "subject", item.msg.Subject)
		return
	}

	item.retries++
	backoff := time.Duration(item.retries) * 5 * time.Second
	slog.Warn("mailer: send failed, retrying with backoff", "to", item.msg.To, "subject", item.msg.Subject, "retry", item.retries, "backoff", backoff)

	go func() {
		select {
		case <-time.After(backoff):
			select {
			case q.ch <- item:
			default:
				slog.Error("mailer: requeue failed, queue full, message dropped", "to", item.msg.To)
			}
		case <-ctx.Done():
			slog.Warn("mailer: retry cancelled during shutdown", "to", item.msg.To)
		}
	}()
}

// drain flushes remaining queued messages on shutdown, best-effort.
func (q *Queue) drain() {
	for {
		select {
		case item := <-q.ch:
			if err := q.mailer.send(item.msg); err != nil {
				slog.Error("mailer: drain send failed", "to", item.msg.To, "err", err)
			}
		default:
			return
		}
	}
}

// SendReport encrypts body then enqueues the encrypted message.
// Implements ReportSender.
func (q *Queue) SendReport(body string) error {
	q.mailer.mu.RLock()
	cfg := q.mailer.cfg
	q.mailer.mu.RUnlock()

	if cfg.PGPPublicKey == "" {
		return fmt.Errorf("PGP public key is not configured")
	}

	encrypted, err := encryptBody(cfg.PGPPublicKey, body)
	if err != nil {
		return fmt.Errorf("encrypt report: %w", err)
	}

	return q.Enqueue(Message{
		To:      cfg.To,
		Subject: "Report from Firewatch",
		Body:    encrypted,
		IsHTML:  false,
	})
}

// SendInvite constructs an invite email then enqueues it.
func (q *Queue) SendInvite(to, inviteURL string) error {
	return q.Enqueue(Message{
		To:      []string{to},
		Subject: "You've been invited to Firewatch",
		Body: fmt.Sprintf(
			"You have been invited to access Firewatch.\n\nAccept your invitation:\n%s\n\nThis link expires in 48 hours.",
			inviteURL,
		),
		IsHTML:  true,
	})
}

// Ping delegates to the underlying Mailer.
func (q *Queue) Ping() error {
	return q.mailer.Ping()
}

func (q *Queue) Reconfigure(cfg *Config) {
	q.mailer.Reconfigure(cfg)
}

// CanEncrypt delegates to the underlying Mailer.
// Implements ReportSender.
func (q *Queue) CanEncrypt() error {
	return q.mailer.CanEncrypt()
}
