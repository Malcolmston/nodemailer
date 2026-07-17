package nodemailer

import (
	"bufio"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// richSMTPServer is a configurable in-process SMTP server used to exercise
// authentication, DSN parameters, connection reuse and verification.
type richSMTPServer struct {
	ln net.Listener

	advertiseAuth     bool // advertise AUTH XOAUTH2 in EHLO
	advertiseStartTLS bool // advertise STARTTLS (never actually upgraded here)
	failAuth          bool // reject AUTH with 535

	mu           sync.Mutex
	conns        int      // total accepted connections
	messages     int      // total messages accepted (DATA completed)
	authUser     string   // decoded XOAUTH2 user
	authToken    string   // decoded XOAUTH2 token
	lastMailLine string   // full MAIL FROM line
	lastRcptLine string   // last RCPT TO line
	datas        []string // captured message bodies
}

func newRichServer(t *testing.T) *richSMTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &richSMTPServer{ln: ln}
	go s.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return s
}

func (s *richSMTPServer) hostPort(t *testing.T) (string, int) {
	t.Helper()
	host, port, err := net.SplitHostPort(s.ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	n, _ := strconv.Atoi(port)
	return host, n
}

func (s *richSMTPServer) transport(t *testing.T) *SMTPTransport {
	host, port := s.hostPort(t)
	return &SMTPTransport{Host: host, Port: port, LocalName: "test.local"}
}

func (s *richSMTPServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.conns++
		s.mu.Unlock()
		go s.handle(conn)
	}
}

func (s *richSMTPServer) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	write := func(line string) {
		_, _ = w.WriteString(line + "\r\n")
		_ = w.Flush()
	}
	write("220 rich ESMTP")

	var data strings.Builder
	inData := false
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		if inData {
			if strings.TrimRight(line, "\r\n") == "." {
				inData = false
				s.mu.Lock()
				s.messages++
				s.datas = append(s.datas, data.String())
				s.mu.Unlock()
				data.Reset()
				write("250 OK queued")
				continue
			}
			data.WriteString(line)
			continue
		}
		trimmed := strings.TrimRight(line, "\r\n")
		upper := strings.ToUpper(trimmed)
		switch {
		case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
			write("250-rich greets you")
			if s.advertiseStartTLS {
				write("250-STARTTLS")
			}
			if s.advertiseAuth {
				write("250-AUTH XOAUTH2 PLAIN")
			}
			write("250 SIZE 10485760")
		case strings.HasPrefix(upper, "AUTH XOAUTH2"):
			s.handleAuth(trimmed, r, write)
		case strings.HasPrefix(upper, "MAIL FROM:"):
			s.mu.Lock()
			s.lastMailLine = trimmed
			s.mu.Unlock()
			write("250 OK")
		case strings.HasPrefix(upper, "RCPT TO:"):
			s.mu.Lock()
			s.lastRcptLine = trimmed
			s.mu.Unlock()
			write("250 OK")
		case upper == "DATA":
			inData = true
			write("354 End data with <CRLF>.<CRLF>")
		case upper == "RSET":
			write("250 OK")
		case upper == "QUIT":
			write("221 Bye")
			return
		default:
			write("250 OK")
		}
	}
}

func (s *richSMTPServer) handleAuth(line string, r *bufio.Reader, write func(string)) {
	// The initial response may be on the AUTH line ("AUTH XOAUTH2 <b64>").
	fields := strings.Fields(line)
	var resp string
	if len(fields) >= 3 {
		resp = fields[2]
	} else {
		write("334 ")
		l, err := r.ReadString('\n')
		if err != nil {
			return
		}
		resp = strings.TrimRight(l, "\r\n")
	}
	if user, token, ok := decodeXOAuth2([]byte(resp)); ok {
		s.mu.Lock()
		s.authUser = user
		s.authToken = token
		s.mu.Unlock()
	}
	if s.failAuth {
		// Send a base64 error challenge, read the client's empty ack, then fail.
		write("334 eyJzdGF0dXMiOiI0MDEifQ==")
		_, _ = r.ReadString('\n')
		write("535 5.7.8 Authentication failed")
		return
	}
	write("235 2.7.0 Accepted")
}
