package nodemailer

import (
	"bufio"
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestMemoryTransport(t *testing.T) {
	mt := &MemoryTransport{}
	if _, ok := mt.Last(); ok {
		t.Error("empty transport should have no last message")
	}
	if err := mt.Send("a@b.com", []string{"c@d.com"}, []byte("raw")); err != nil {
		t.Fatal(err)
	}
	last, ok := mt.Last()
	if !ok {
		t.Fatal("expected a captured message")
	}
	if last.From != "a@b.com" || last.To[0] != "c@d.com" || string(last.Raw) != "raw" {
		t.Errorf("unexpected capture: %+v", last)
	}
}

func TestJSONTransport(t *testing.T) {
	jt := &JSONTransport{}
	if _, ok := jt.Last(); ok {
		t.Error("expected no records")
	}
	if err := jt.Send("a@b.com", []string{"c@d.com"}, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	rec, ok := jt.Last()
	if !ok {
		t.Fatal("expected a record")
	}
	var got jsonRecord
	if err := json.Unmarshal(rec, &got); err != nil {
		t.Fatal(err)
	}
	if got.From != "a@b.com" || got.To[0] != "c@d.com" {
		t.Errorf("bad record: %+v", got)
	}
	if got.Raw != "aGVsbG8=" { // base64 of "hello"
		t.Errorf("raw = %q", got.Raw)
	}
}

func TestSMTPTransportSendError(t *testing.T) {
	if err := (&SMTPTransport{}).Send("a@b.com", []string{"c@d.com"}, nil); err == nil {
		t.Error("expected error when Host is empty")
	}
}

// fakeSMTPServer is a minimal SMTP server used to exercise SMTPTransport over a
// real socket without TLS.
type fakeSMTPServer struct {
	ln   net.Listener
	mu   sync.Mutex
	from string
	to   []string
	data string
}

func newFakeSMTPServer(t *testing.T) *fakeSMTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &fakeSMTPServer{ln: ln}
	go s.serve()
	return s
}

func (s *fakeSMTPServer) addr() string { return s.ln.Addr().String() }

func (s *fakeSMTPServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *fakeSMTPServer) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	write := func(line string) {
		_, _ = w.WriteString(line + "\r\n")
		_ = w.Flush()
	}
	write("220 fake ESMTP")
	inData := false
	var data strings.Builder
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		if inData {
			if strings.TrimRight(line, "\r\n") == "." {
				inData = false
				s.mu.Lock()
				s.data = data.String()
				s.mu.Unlock()
				write("250 OK queued")
				continue
			}
			data.WriteString(line)
			continue
		}
		cmd := strings.ToUpper(strings.TrimRight(line, "\r\n"))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			write("250-fake greets you")
			write("250 OK")
		case strings.HasPrefix(cmd, "MAIL FROM:"):
			s.mu.Lock()
			s.from = extractAngle(line[len("MAIL FROM:"):])
			s.mu.Unlock()
			write("250 OK")
		case strings.HasPrefix(cmd, "RCPT TO:"):
			s.mu.Lock()
			s.to = append(s.to, extractAngle(line[len("RCPT TO:"):]))
			s.mu.Unlock()
			write("250 OK")
		case cmd == "DATA":
			inData = true
			write("354 End data with <CRLF>.<CRLF>")
		case cmd == "QUIT":
			write("221 Bye")
			return
		default:
			write("250 OK")
		}
	}
}

func extractAngle(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "<"); i >= 0 {
		if j := strings.Index(s, ">"); j > i {
			return s[i+1 : j]
		}
	}
	return s
}

func TestSMTPTransportAgainstFakeServer(t *testing.T) {
	srv := newFakeSMTPServer(t)
	defer func() { _ = srv.ln.Close() }()

	host, port, err := net.SplitHostPort(srv.addr())
	if err != nil {
		t.Fatal(err)
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}

	tr := &SMTPTransport{Host: host, Port: portNum, LocalName: "test.local"}
	raw := []byte("Subject: Hi\r\n\r\nbody\r\n")
	if err := tr.Send("ada@example.com", []string{"grace@example.com", "carl@example.com"}, raw); err != nil {
		t.Fatalf("Send: %v", err)
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.from != "ada@example.com" {
		t.Errorf("MAIL FROM = %q", srv.from)
	}
	if len(srv.to) != 2 || srv.to[0] != "grace@example.com" || srv.to[1] != "carl@example.com" {
		t.Errorf("RCPT TO = %v", srv.to)
	}
	if !strings.Contains(srv.data, "Subject: Hi") || !strings.Contains(srv.data, "body") {
		t.Errorf("DATA = %q", srv.data)
	}
}

func TestSMTPTransportConfigHelpers(t *testing.T) {
	tr := &SMTPTransport{Host: "smtp.example.com"}
	if tr.address() != "smtp.example.com:25" {
		t.Errorf("default address = %q", tr.address())
	}
	tr.Port = 587
	if tr.address() != "smtp.example.com:587" {
		t.Errorf("address = %q", tr.address())
	}
	if tr.auth() != nil {
		t.Error("expected no auth without username")
	}
	tr.Username = "u"
	tr.Password = "p"
	if tr.auth() == nil {
		t.Error("expected PLAIN auth with username")
	}
	if cfg := tr.tlsConfig(); cfg.ServerName != "smtp.example.com" {
		t.Errorf("tlsConfig ServerName = %q", cfg.ServerName)
	}
}
