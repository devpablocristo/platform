package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"os"
	"strconv"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

type EmailMessage struct {
	To       string
	Subject  string
	HTMLBody string
	TextBody string
}

type EmailSender interface {
	Send(ctx context.Context, message EmailMessage) error
}

type noopEmailSender struct {
	logger *slog.Logger
}

func NewNoopEmailSender(logger *slog.Logger) EmailSender {
	if logger == nil {
		logger = slog.Default()
	}
	return &noopEmailSender{logger: logger}
}

func (s *noopEmailSender) Send(_ context.Context, message EmailMessage) error {
	s.logger.Info("noop email sender", "to", strings.TrimSpace(message.To), "subject", strings.TrimSpace(message.Subject))
	return nil
}

type smtpSender struct {
	host     string
	port     int
	user     string
	password string
	from     string
	sendMail func(string, smtp.Auth, string, []string, []byte) error
}

func NewSMTPSender(host string, port int, user, password, from string) EmailSender {
	return &smtpSender{
		host:     strings.TrimSpace(host),
		port:     port,
		user:     strings.TrimSpace(user),
		password: password,
		from:     strings.TrimSpace(from),
		sendMail: smtp.SendMail,
	}
}

func (s *smtpSender) Send(_ context.Context, message EmailMessage) error {
	if strings.TrimSpace(message.To) == "" {
		return fmt.Errorf("email recipient is required")
	}
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	boundary := "core-notifications-boundary"
	payload := strings.Join([]string{
		"From: " + s.from,
		"To: " + strings.TrimSpace(message.To),
		"Subject: " + strings.TrimSpace(message.Subject),
		"MIME-Version: 1.0",
		"Content-Type: multipart/alternative; boundary=" + boundary,
		"",
		"--" + boundary,
		"Content-Type: text/plain; charset=UTF-8",
		"",
		message.TextBody,
		"--" + boundary,
		"Content-Type: text/html; charset=UTF-8",
		"",
		message.HTMLBody,
		"--" + boundary + "--",
	}, "\r\n")
	var auth smtp.Auth
	if s.user != "" {
		auth = smtp.PlainAuth("", s.user, s.password, s.host)
	}
	if err := s.sendMail(addr, auth, s.from, []string{strings.TrimSpace(message.To)}, []byte(payload)); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

type sesClient interface {
	SendEmail(context.Context, *sesv2.SendEmailInput, ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}

type sesSender struct {
	client sesClient
	from   string
}

func NewSESSender(client *sesv2.Client, from string) EmailSender {
	return &sesSender{client: client, from: strings.TrimSpace(from)}
}

func (s *sesSender) Send(ctx context.Context, message EmailMessage) error {
	to := strings.TrimSpace(message.To)
	if to == "" {
		return fmt.Errorf("email recipient is required")
	}
	subject := strings.TrimSpace(message.Subject)
	if _, err := s.client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: &s.from,
		Destination: &sestypes.Destination{
			ToAddresses: []string{to},
		},
		Content: &sestypes.EmailContent{
			Simple: &sestypes.Message{
				Subject: &sestypes.Content{Data: &subject},
				Body: &sestypes.Body{
					Text: &sestypes.Content{Data: &message.TextBody},
					Html: &sestypes.Content{Data: &message.HTMLBody},
				},
			},
		},
	}); err != nil {
		return fmt.Errorf("ses send: %w", err)
	}
	return nil
}

type EmailConfig struct {
	Backend   string
	Host      string
	Port      int
	User      string
	Password  string
	From      string
	AWSRegion string
}

func EmailConfigFromEnv(prefix string) EmailConfig {
	prefix = normalizePrefix(prefix)
	return EmailConfig{
		Backend:   firstNonEmpty(os.Getenv(prefix+"BACKEND"), os.Getenv("NOTIFICATION_BACKEND"), "noop"),
		Host:      firstNonEmpty(os.Getenv(prefix+"SMTP_HOST"), os.Getenv("SMTP_HOST"), "localhost"),
		Port:      intFromEnv(firstNonEmpty(os.Getenv(prefix+"SMTP_PORT"), os.Getenv("SMTP_PORT")), 1025),
		User:      firstNonEmpty(os.Getenv(prefix+"SMTP_USER"), os.Getenv("SMTP_USER")),
		Password:  firstNonEmpty(os.Getenv(prefix+"SMTP_PASSWORD"), os.Getenv("SMTP_PASSWORD")),
		From:      firstNonEmpty(os.Getenv(prefix+"FROM_EMAIL"), os.Getenv("SMTP_FROM_EMAIL"), os.Getenv("AWS_SES_FROM_EMAIL"), "noreply@example.com"),
		AWSRegion: firstNonEmpty(os.Getenv(prefix+"AWS_REGION"), os.Getenv("AWS_REGION"), "us-east-1"),
	}
}

func NewEmailSender(ctx context.Context, config EmailConfig, logger *slog.Logger) (EmailSender, error) {
	switch strings.ToLower(strings.TrimSpace(config.Backend)) {
	case "", "noop":
		return NewNoopEmailSender(logger), nil
	case "smtp":
		return NewSMTPSender(config.Host, config.Port, config.User, config.Password, config.From), nil
	case "ses":
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(config.AWSRegion))
		if err != nil {
			return nil, fmt.Errorf("load aws config: %w", err)
		}
		return NewSESSender(sesv2.NewFromConfig(awsCfg), config.From), nil
	default:
		return nil, fmt.Errorf("unsupported notification backend: %s", config.Backend)
	}
}

func normalizePrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return ""
	}
	return strings.TrimSuffix(prefix, "_") + "_"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func intFromEnv(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
