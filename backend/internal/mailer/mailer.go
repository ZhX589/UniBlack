package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
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
	Timeout            time.Duration // whole SMTP exchange; default 15s
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
	for _, field := range append([]string{from, msg.Subject}, msg.To...) {
		if strings.ContainsAny(field, "\r\n") {
			return fmt.Errorf("invalid mail header")
		}
	}
	addr := net.JoinHostPort(m.Config.Host, fmt.Sprint(m.Config.Port))

	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(msg.To, ",") + "\r\n")
	b.WriteString("Subject: " + msg.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(msg.Body)
	raw := []byte(b.String())

	return m.send(ctx, addr, from, msg.To, raw)
}

func (m SMTPMailer) send(ctx context.Context, addr, from string, to []string, raw []byte) (sendErr error) {
	timeout := m.Config.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	defer func() {
		if ctx.Err() != nil {
			sendErr = ctx.Err()
		}
	}()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	// Deadlines bound the entire exchange, including greeting, AUTH and DATA.
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return err
		}
	}
	rawConn := conn
	stop := context.AfterFunc(ctx, func() { _ = rawConn.Close() })
	defer stop()
	tlsConfig := &tls.Config{
		ServerName:         m.Config.Host,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: m.Config.InsecureSkipVerify, //nolint:gosec // configurable for self-signed
	}
	if m.Config.SSL {
		tlsConn := tls.Client(conn, tlsConfig)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			return err
		}
		conn = tlsConn
	}
	c, err := smtp.NewClient(conn, m.Config.Host)
	if err != nil {
		return err
	}
	defer c.Close()
	if !m.Config.SSL {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(tlsConfig); err != nil {
				return err
			}
		}
	}
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
