package nodemailer

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"sync"
)

// Transport delivers a fully-encoded message to its recipients. The raw
// argument is the RFC 5322 bytes produced by Message.Build.
type Transport interface {
	Send(from string, to []string, raw []byte) error
}

// MemoryTransport is an in-memory Transport that captures every message it is
// asked to send. It is safe for concurrent use and is intended for tests.
type MemoryTransport struct {
	mu       sync.Mutex
	Messages []CapturedMessage
}

// CapturedMessage is a single message recorded by a MemoryTransport.
type CapturedMessage struct {
	From string
	To   []string
	Raw  []byte
}

// Send records the message in memory. It never fails.
func (t *MemoryTransport) Send(from string, to []string, raw []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	cp := make([]byte, len(raw))
	copy(cp, raw)
	t.Messages = append(t.Messages, CapturedMessage{
		From: from,
		To:   append([]string(nil), to...),
		Raw:  cp,
	})
	return nil
}

// Last returns the most recently captured message and true, or a zero value
// and false when nothing has been sent.
func (t *MemoryTransport) Last() (CapturedMessage, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.Messages) == 0 {
		return CapturedMessage{}, false
	}
	return t.Messages[len(t.Messages)-1], true
}

// JSONTransport is a Transport that captures a JSON serialisation of each
// message instead of delivering it, mirroring nodemailer's jsonTransport. The
// raw body is base64-encoded so it round-trips through JSON safely.
type JSONTransport struct {
	mu      sync.Mutex
	Records []json.RawMessage
}

// jsonRecord is the on-the-wire shape produced by JSONTransport.
type jsonRecord struct {
	From string   `json:"from"`
	To   []string `json:"to"`
	Raw  string   `json:"raw"` // base64 std encoding
}

// Send serialises the message to JSON and stores it.
func (t *JSONTransport) Send(from string, to []string, raw []byte) error {
	rec := jsonRecord{
		From: from,
		To:   append([]string(nil), to...),
		Raw:  base64.StdEncoding.EncodeToString(raw),
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Records = append(t.Records, json.RawMessage(b))
	return nil
}

// Last returns the most recent JSON record and true, or nil and false.
func (t *JSONTransport) Last() (json.RawMessage, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.Records) == 0 {
		return nil, false
	}
	return t.Records[len(t.Records)-1], true
}

// SMTPTransport delivers messages over SMTP using the standard net/smtp client.
// It supports plain SMTP, implicit TLS (SMTPS) and STARTTLS with optional
// authentication.
type SMTPTransport struct {
	// Host is the SMTP server hostname.
	Host string
	// Port is the SMTP server port (e.g. 25, 465, 587).
	Port int
	// Username and Password enable PLAIN authentication when Username is set
	// and Auth is nil.
	Username string
	Password string
	// Auth overrides the default PLAIN authentication mechanism.
	Auth smtp.Auth
	// TLS uses implicit TLS for the whole connection (typically port 465).
	TLS bool
	// STARTTLS upgrades a plaintext connection to TLS (typically port 587).
	STARTTLS bool
	// TLSConfig is the TLS configuration used for TLS/STARTTLS. When nil a
	// config with ServerName set to Host is used.
	TLSConfig *tls.Config
	// LocalName is the name sent in the EHLO/HELO command.
	LocalName string
	// dial lets tests inject a custom dialer; nil uses net.Dial.
	dial func(network, addr string) (net.Conn, error)
}

// address returns the host:port dial target.
func (t *SMTPTransport) address() string {
	port := t.Port
	if port == 0 {
		port = 25
	}
	return net.JoinHostPort(t.Host, strconv.Itoa(port))
}

func (t *SMTPTransport) tlsConfig() *tls.Config {
	if t.TLSConfig != nil {
		return t.TLSConfig
	}
	return &tls.Config{ServerName: t.Host}
}

// Send delivers the raw message to the SMTP server.
func (t *SMTPTransport) Send(from string, to []string, raw []byte) error {
	if t.Host == "" {
		return fmt.Errorf("nodemailer: SMTPTransport requires a Host")
	}
	conn, err := t.connect()
	if err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, t.Host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer func() { _ = client.Close() }()

	if t.LocalName != "" {
		if err := client.Hello(t.LocalName); err != nil {
			return err
		}
	}

	if t.STARTTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(t.tlsConfig()); err != nil {
				return err
			}
		}
	}

	if auth := t.auth(); auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}

	if err := client.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(raw); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// connect dials the server, wrapping the connection in TLS for implicit TLS.
func (t *SMTPTransport) connect() (net.Conn, error) {
	dial := t.dial
	if dial == nil {
		dial = net.Dial
	}
	conn, err := dial("tcp", t.address())
	if err != nil {
		return nil, err
	}
	if t.TLS {
		return tls.Client(conn, t.tlsConfig()), nil
	}
	return conn, nil
}

// auth resolves the authentication mechanism to use, if any.
func (t *SMTPTransport) auth() smtp.Auth {
	if t.Auth != nil {
		return t.Auth
	}
	if t.Username != "" {
		return smtp.PlainAuth("", t.Username, t.Password, t.Host)
	}
	return nil
}
