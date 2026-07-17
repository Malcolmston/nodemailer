package nodemailer

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
	"time"
)

var fixedDate = time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)

func newFixed() *Message {
	return New().
		SetFrom("Ada Lovelace <ada@example.com>").
		AddTo("Grace Hopper <grace@example.com>").
		SetDate(fixedDate).
		SetMessageID("id@example.com").
		SetBoundary("B")
}

func TestBuildTextOnlyExact(t *testing.T) {
	m := New().
		SetFrom("ada@example.com").
		AddTo("grace@example.com").
		SetSubject("Hi").
		SetText("Hello world").
		SetDate(fixedDate).
		SetMessageID("id@example.com")
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	want := "Date: Fri, 02 Jan 2026 15:04:05 +0000\r\n" +
		"From: <ada@example.com>\r\n" +
		"To: <grace@example.com>\r\n" +
		"Message-ID: <id@example.com>\r\n" +
		"Subject: Hi\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"\r\n" +
		"Hello world"
	if string(raw) != want {
		t.Errorf("text-only mismatch:\n got %q\nwant %q", raw, want)
	}
}

func TestBuildAlternativeExact(t *testing.T) {
	m := newFixed().
		SetSubject("Hi").
		SetText("Hello").
		SetHTML("<p>Hi</p>")
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	want := "Date: Fri, 02 Jan 2026 15:04:05 +0000\r\n" +
		"From: \"Ada Lovelace\" <ada@example.com>\r\n" +
		"To: \"Grace Hopper\" <grace@example.com>\r\n" +
		"Message-ID: <id@example.com>\r\n" +
		"Subject: Hi\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/alternative; boundary=\"B\"\r\n" +
		"\r\n" +
		"--B\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"\r\n" +
		"Hello\r\n" +
		"--B\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"\r\n" +
		"<p>Hi</p>\r\n" +
		"--B--\r\n"
	if string(raw) != want {
		t.Errorf("alternative mismatch:\n got %q\nwant %q", raw, want)
	}
}

func TestSubjectEncodedWord(t *testing.T) {
	m := newFixed().SetSubject("Héllo").SetText("x")
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("Subject: =?utf-8?q?H=C3=A9llo?=")) {
		t.Errorf("subject not RFC2047-encoded:\n%s", raw)
	}
}

func TestAttachmentBase64(t *testing.T) {
	content := bytes.Repeat([]byte("nodemailer "), 10) // > 76 chars encoded
	m := newFixed().
		SetText("see attachment").
		AttachBytes("data.bin", "application/octet-stream", content)
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("Content-Type: multipart/mixed;")) {
		t.Error("expected multipart/mixed wrapper")
	}
	if !bytes.Contains(raw, []byte(`Content-Disposition: attachment; filename="data.bin"`)) {
		t.Error("missing attachment disposition")
	}
	// Extract the base64 block and confirm it decodes to the content and is
	// wrapped at 76 columns.
	part := extractPartBody(t, raw, "data.bin")
	for _, line := range strings.Split(strings.TrimRight(part, "\r\n"), "\r\n") {
		if len(line) > 76 {
			t.Errorf("base64 line exceeds 76 chars: %d", len(line))
		}
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(part, "\r\n", ""))
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	if !bytes.Equal(decoded, content) {
		t.Errorf("decoded attachment mismatch")
	}
}

func TestInlineImageRelated(t *testing.T) {
	m := newFixed().
		SetHTML(`<img src="cid:logo">`).
		Embed("logo", "logo.png", "image/png", []byte{0x89, 0x50})
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "Content-Type: multipart/related;") {
		t.Error("expected multipart/related for inline image")
	}
	if !strings.Contains(s, "Content-ID: <logo>") {
		t.Error("expected Content-ID header")
	}
	if !strings.Contains(s, "Content-Disposition: inline; filename=\"logo.png\"") {
		t.Error("expected inline disposition")
	}
	if strings.Contains(s, "multipart/mixed") {
		t.Error("no regular attachments, should not use multipart/mixed")
	}
}

// TestParseRoundTrip confirms the output is parseable by the standard library
// mail and multipart readers.
func TestParseRoundTrip(t *testing.T) {
	m := newFixed().
		SetSubject("Round trip").
		AddCc("carl@example.com").
		AddReplyTo("reply@example.com").
		SetText("plain body").
		SetHTML("<p>html body</p>").
		AttachBytes("a.txt", "text/plain", []byte("attached")).
		Embed("img1", "p.png", "image/png", []byte{1, 2, 3})
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}

	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("mail.ReadMessage: %v", err)
	}
	if got := msg.Header.Get("Subject"); got != "Round trip" {
		t.Errorf("Subject = %q", got)
	}
	if got := msg.Header.Get("Cc"); !strings.Contains(got, "carl@example.com") {
		t.Errorf("Cc = %q", got)
	}
	if got := msg.Header.Get("Reply-To"); !strings.Contains(got, "reply@example.com") {
		t.Errorf("Reply-To = %q", got)
	}

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	if mediaType != "multipart/mixed" {
		t.Fatalf("top media type = %q", mediaType)
	}
	countParts(t, msg.Body, params["boundary"])
}

// countParts walks the multipart tree to ensure every boundary is balanced and
// readable.
func countParts(t *testing.T, r io.Reader, boundary string) {
	t.Helper()
	mr := multipart.NewReader(r, boundary)
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Fatalf("NextPart: %v", err)
		}
		ct := p.Header.Get("Content-Type")
		if mt, params, err := mime.ParseMediaType(ct); err == nil && strings.HasPrefix(mt, "multipart/") {
			countParts(t, p, params["boundary"])
		} else if _, err := io.ReadAll(p); err != nil {
			t.Fatalf("read part: %v", err)
		}
	}
}

func TestBuildErrors(t *testing.T) {
	if _, err := New().SetFrom("bad").Build(); err == nil {
		t.Error("expected error from invalid From")
	}
	if _, err := New().SetFrom("a@b.com").SetText("x").Build(); err == nil {
		t.Error("expected error for no recipients")
	}
	if _, err := New().SetFrom("a@b.com").AddTo("c@d.com").Build(); err == nil {
		t.Error("expected error for no content")
	}
	if _, err := (&Message{}).Build(); err == nil {
		t.Error("expected error for empty message")
	}
}

func TestDeterministic(t *testing.T) {
	build := func() []byte {
		raw, err := newFixed().SetText("hi").SetHTML("<p>hi</p>").
			AttachBytes("x.txt", "text/plain", []byte("y")).Build()
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	if !bytes.Equal(build(), build()) {
		t.Error("output is not deterministic for fixed boundary/date/id")
	}
}

func TestHeaderFolding(t *testing.T) {
	m := newFixed().SetText("x").AddHeader("X-Test", "v")
	for i := 0; i < 12; i++ {
		m.AddCc(fmt.Sprintf("recipient-number-%d@longdomain.example.com", i))
	}
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("X-Test: v\r\n")) {
		t.Error("custom header missing")
	}
	// Every physical line must be at most 78 chars (+CRLF); folded lines start
	// with a single space continuation.
	sawFold := false
	for _, line := range strings.Split(string(raw), "\r\n") {
		if len(line) > 78 {
			t.Errorf("unfolded line exceeds 78 chars (%d): %q", len(line), line)
		}
		if strings.HasPrefix(line, " ") && strings.Contains(line, "recipient-number-") {
			sawFold = true
		}
	}
	if !sawFold {
		t.Error("expected the long Cc header to be folded")
	}
	// The folded header must still parse back to all 12 recipients.
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	list, err := msg.Header.AddressList("Cc")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 12 {
		t.Errorf("parsed %d Cc addresses, want 12", len(list))
	}
}

func TestNonASCIIFilename(t *testing.T) {
	m := newFixed().SetText("x").
		AttachBytes("résumé.txt", "text/plain", []byte("hi"))
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	// Non-ASCII filenames use RFC 2047 B-encoding.
	if !bytes.Contains(raw, []byte("=?utf-8?b?")) {
		t.Errorf("expected encoded filename in:\n%s", raw)
	}
}

func TestContentTypeGuessedFromExtension(t *testing.T) {
	m := newFixed().SetText("x").
		Attach(Attachment{Filename: "page.html", Content: []byte("<b>x</b>")})
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("Content-Type: text/html")) {
		t.Errorf("content type not guessed from extension:\n%s", raw)
	}
}

func TestFoldHeaderUnit(t *testing.T) {
	short := foldHeader("Subject", "hello")
	if short != "Subject: hello" {
		t.Errorf("short header changed: %q", short)
	}
	long := foldHeader("X-Long", strings.Repeat("word ", 40))
	for _, line := range strings.Split(long, "\r\n") {
		if len(line) > 78 {
			t.Errorf("folded line too long: %d", len(line))
		}
	}
}

// extractPartBody returns the raw (still-encoded) body bytes of the part whose
// headers mention filename.
func extractPartBody(t *testing.T, raw []byte, filename string) string {
	t.Helper()
	s := string(raw)
	marker := "filename=\"" + filename + "\""
	i := strings.Index(s, marker)
	if i < 0 {
		t.Fatalf("filename %q not found", filename)
	}
	rest := s[i:]
	blank := strings.Index(rest, "\r\n\r\n")
	if blank < 0 {
		t.Fatal("no header/body separator")
	}
	body := rest[blank+4:]
	end := strings.Index(body, "\r\n--")
	if end < 0 {
		t.Fatal("no closing boundary")
	}
	return body[:end]
}
