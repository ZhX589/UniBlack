package mailer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

func smtpFixture(t *testing.T, server func(net.Conn)) SMTPMailer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err == nil {
			defer conn.Close()
			server(conn)
		}
	}()
	t.Cleanup(func() {
		listener.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("SMTP fixture did not stop")
		}
	})
	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	return SMTPMailer{Config: Config{Host: host, Port: p, From: "from@example.test", Timeout: time.Second}}
}

func TestSMTPCancellationInterruptsGreetingAndTLS(t *testing.T) {
	for _, ssl := range []bool{false, true} {
		t.Run(fmt.Sprint(ssl), func(t *testing.T) {
			accepted := make(chan struct{})
			m := smtpFixture(t, func(conn net.Conn) {
				close(accepted)
				buf := make([]byte, 4096)
				for {
					if _, err := conn.Read(buf); err != nil {
						return
					}
				}
			})
			m.Config.SSL = ssl
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- m.Send(ctx, Message{To: []string{"to@example.test"}, Subject: "test", Body: "body"}) }()
			<-accepted
			cancel()
			select {
			case err := <-result:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("error=%v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("cancel did not interrupt SMTP")
			}
		})
	}
}

func TestSMTPDeadlineBoundsGreeting(t *testing.T) {
	m := smtpFixture(t, func(conn net.Conn) { buf := make([]byte, 1); _, _ = conn.Read(buf) })
	m.Config.Timeout = 40 * time.Millisecond
	err := m.Send(context.Background(), Message{To: []string{"to@example.test"}, Subject: "test", Body: "body"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%v", err)
	}
}

func TestSMTPFullExchangeAndCancellationDuringData(t *testing.T) {
	for _, cancelData := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelData), func(t *testing.T) {
			dataReady := make(chan struct{})
			received := make(chan string, 1)
			m := smtpFixture(t, func(conn net.Conn) {
				_, _ = fmt.Fprint(conn, "220 fixture ESMTP\r\n")
				r := bufio.NewReader(conn)
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					switch {
					case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
						_, _ = fmt.Fprint(conn, "250 fixture\r\n")
					case strings.HasPrefix(line, "MAIL"), strings.HasPrefix(line, "RCPT"):
						_, _ = fmt.Fprint(conn, "250 OK\r\n")
					case strings.HasPrefix(line, "DATA"):
						_, _ = fmt.Fprint(conn, "354 continue\r\n")
						var body strings.Builder
						for {
							l, err := r.ReadString('\n')
							if err != nil {
								return
							}
							if l == ".\r\n" {
								break
							}
							body.WriteString(l)
						}
						received <- body.String()
						close(dataReady)
						if cancelData {
							_, _ = r.ReadString('\n')
							return
						}
						_, _ = fmt.Fprint(conn, "250 delivered\r\n")
					case strings.HasPrefix(line, "QUIT"):
						_, _ = fmt.Fprint(conn, "221 bye\r\n")
						return
					default:
						return
					}
				}
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() {
				result <- m.Send(ctx, Message{To: []string{"to@example.test"}, Subject: "test", Body: "proof"})
			}()
			<-dataReady
			if cancelData {
				cancel()
			}
			select {
			case err := <-result:
				if cancelData {
					if !errors.Is(err, context.Canceled) {
						t.Fatalf("error=%v", err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("SMTP exchange did not terminate")
			}
			if body := <-received; !strings.Contains(body, "Subject: test\r\n") || !strings.Contains(body, "proof") {
				t.Fatalf("body=%q", body)
			}
		})
	}
}

func TestSMTPRejectsHeaderInjectionBeforeDial(t *testing.T) {
	m := SMTPMailer{Config: Config{From: "from@example.test"}}
	for _, msg := range []Message{{Subject: "test\r\nBcc: other@example.test"}, {To: []string{"to@example.test\nRCPT"}}} {
		if err := m.Send(context.Background(), msg); err == nil || err.Error() != "invalid mail header" {
			t.Fatalf("error=%v", err)
		}
	}
}
