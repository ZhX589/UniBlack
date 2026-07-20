package mailer

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// Message is a simple outbound email.
type Message struct {
	To      []string
	Subject string
	Body    string
}

// Mailer sends email. Implementations are selected from system settings.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

// Config is SMTP configuration loaded from system_settings / env.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// New builds a mailer. When host is empty, returns LogMailer (dev-friendly).
func New(cfg Config) Mailer {
	if cfg.Host == "" {
		return LogMailer{}
	}
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	return SMTPMailer{Config: cfg}
}

// LogMailer logs messages instead of sending (default for unconfigured SMTP).
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, msg Message) error {
	fmt.Printf("[mailer] to=%v subject=%q body=%q\n", msg.To, msg.Subject, msg.Body)
	return nil
}

// SMTPMailer sends via net/smtp.
type SMTPMailer struct {
	Config Config
}

func (m SMTPMailer) Send(_ context.Context, msg Message) error {
	from := m.Config.From
	if from == "" {
		from = m.Config.Username
	}
	if from == "" {
		return fmt.Errorf("smtp from address not configured")
	}
	addr := fmt.Sprintf("%s:%d", m.Config.Host, m.Config.Port)
	var auth smtp.Auth
	if m.Config.Username != "" {
		auth = smtp.PlainAuth("", m.Config.Username, m.Config.Password, m.Config.Host)
	}
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(msg.To, ",") + "\r\n")
	b.WriteString("Subject: " + msg.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(msg.Body)
	return smtp.SendMail(addr, auth, from, msg.To, []byte(b.String()))
}
