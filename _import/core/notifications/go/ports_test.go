package notifications

import (
	"context"
	"errors"
	"log/slog"
	"net/smtp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
)

func TestEmailConfigFromEnv(t *testing.T) {
	t.Setenv("MAIL_BACKEND", "smtp")
	t.Setenv("MAIL_SMTP_HOST", "smtp.example.com")
	t.Setenv("MAIL_SMTP_PORT", "2525")
	t.Setenv("MAIL_FROM_EMAIL", "noreply@example.com")

	config := EmailConfigFromEnv("MAIL")
	if config.Backend != "smtp" {
		t.Fatalf("unexpected backend: %q", config.Backend)
	}
	if config.Port != 2525 {
		t.Fatalf("unexpected port: %d", config.Port)
	}
}

func TestNoopEmailSender(t *testing.T) {
	t.Parallel()

	sender := NewNoopEmailSender(slog.Default())
	if err := sender.Send(context.Background(), EmailMessage{To: "user@example.com", Subject: "Hello"}); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
}

func TestSMTPSenderBuildsMessage(t *testing.T) {
	t.Parallel()

	sender := NewSMTPSender("smtp.example.com", 2525, "user", "secret", "noreply@example.com").(*smtpSender)
	var called bool
	sender.sendMail = func(addr string, _ smtp.Auth, from string, to []string, msg []byte) error {
		called = true
		if addr != "smtp.example.com:2525" {
			t.Fatalf("unexpected addr: %q", addr)
		}
		if from != "noreply@example.com" || len(to) != 1 || to[0] != "user@example.com" {
			t.Fatalf("unexpected envelope: from=%q to=%v", from, to)
		}
		if len(msg) == 0 {
			t.Fatal("expected message bytes")
		}
		return nil
	}

	err := sender.Send(context.Background(), EmailMessage{
		To:       "user@example.com",
		Subject:  "Hello",
		TextBody: "text",
		HTMLBody: "<p>html</p>",
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if !called {
		t.Fatal("expected sendMail call")
	}
}

func TestSESSenderPropagatesError(t *testing.T) {
	t.Parallel()

	sender := &sesSender{
		client: fakeSESClient{err: errors.New("boom")},
		from:   "noreply@example.com",
	}
	err := sender.Send(context.Background(), EmailMessage{To: "user@example.com", Subject: "Hello"})
	if err == nil {
		t.Fatal("expected error")
	}
}

type fakeSESClient struct {
	err error
}

func (c fakeSESClient) SendEmail(context.Context, *sesv2.SendEmailInput, ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &sesv2.SendEmailOutput{}, nil
}
