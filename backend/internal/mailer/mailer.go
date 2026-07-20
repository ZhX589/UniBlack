package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// Message is a simple outbound email.
type Message struct {
	To      []string
	Subject string
	Body    string
}

// Mailer sends email. Implementations are selected from system settings / env.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

// Config is SMTP configuration (NewAPI-style options).
type Config struct {
	Host               string
	Port               int
	Username           string
	Password           string
	From               string
	SSL                bool // implicit TLS (465)
	InsecureSkipVerify bool
}

// New builds a mailer. When host is empty, returns LogMailer (dev-friendly).
func New(cfg Config) Mailer {
	if cfg.Host == "" {
		return LogMailer{}
	}
	if cfg.Port == 0 {
		if cfg.SSL {
			cfg.Port = 465
		} else {
			cfg.Port = 587
		}
	}
	return SMTPMailer{Config: cfg}
}

// LogMailer logs messages instead of sending (default for unconfigured SMTP).
type LogMailer struct{}

func (LogMailer) Send(_ context.Context, msg Message) error {
	fmt.Printf("[mailer] to=%v subject=%q body=%q\n", msg.To, msg.Subject, msg.Body)
	return nil
}

// SMTPMailer sends via net/smtp with optional SSL.
type SMTPMailer struct {
	Config Config
}

func (m SMTPMailer) Send(ctx context.Context, msg Message) error {
	from := m.Config.From
	if from == "" {
		from = m.Config.Username
	}
	if from == "" {
		return fmt.Errorf("smtp from address not configured")
	}
	addr := fmt.Sprintf("%s:%d", m.Config.Host, m.Config.Port)

	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(msg.To, ",") + "\r\n")
	b.WriteString("Subject: " + msg.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(msg.Body)
	raw := []byte(b.String())

	if m.Config.SSL {
		return m.sendSSL(ctx, addr, from, msg.To, raw)
	}
	var auth smtp.Auth
	if m.Config.Username != "" {
		auth = smtp.PlainAuth("", m.Config.Username, m.Config.Password, m.Config.Host)
	}
	// smtp.SendMail does not take context; best-effort
	return smtp.SendMail(addr, auth, from, msg.To, raw)
}

func (m SMTPMailer) sendSSL(ctx context.Context, addr, from string, to []string, raw []byte) error {
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         m.Config.Host,
		InsecureSkipVerify: m.Config.InsecureSkipVerify, //nolint:gosec // configurable for self-signed
	})
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		return err
	}
	c, err := smtp.NewClient(tlsConn, m.Config.Host)
	if err != nil {
		return err
	}
	defer c.Close()
	if m.Config.Username != "" {
		if err := c.Auth(smtp.PlainAuth("", m.Config.Username, m.Config.Password, m.Config.Host)); err != nil {
			return err
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(raw); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
