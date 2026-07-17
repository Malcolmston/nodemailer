package nodemailer

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAttachFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(path, []byte("file contents"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newFixed().SetText("see file").AttachFile(path)
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
	if len(m.Attachments) != 1 {
		t.Fatalf("got %d attachments", len(m.Attachments))
	}
	a := m.Attachments[0]
	if a.Filename != "note.txt" || string(a.Content) != "file contents" {
		t.Errorf("unexpected attachment: %+v", a)
	}
	if a.ContentType != "text/plain" && !strings.HasPrefix(a.ContentType, "text/plain") {
		t.Errorf("content type = %q", a.ContentType)
	}
}

func TestAttachFileMissing(t *testing.T) {
	m := newFixed().SetText("x").AttachFile("/no/such/file/xyz")
	if m.Err() == nil {
		t.Error("expected error for missing file")
	}
}

func TestAttachReader(t *testing.T) {
	m := newFixed().SetText("x").
		AttachReader("data.bin", "", bytes.NewReader([]byte{1, 2, 3, 4}))
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
	a := m.Attachments[0]
	if !bytes.Equal(a.Content, []byte{1, 2, 3, 4}) {
		t.Errorf("content = %v", a.Content)
	}
	if a.ContentType == "" {
		t.Error("content type should be sniffed")
	}
}

func TestEmbedReaderAndFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "logo.png")
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if err := os.WriteFile(path, png, 0o644); err != nil {
		t.Fatal(err)
	}
	m := newFixed().SetHTML(`<img src="cid:logo"><img src="cid:pic">`).
		EmbedFile("logo", path).
		EmbedReader("pic", "pic.gif", "image/gif", bytes.NewReader([]byte("GIF89a")))
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
	if len(m.Attachments) != 2 {
		t.Fatalf("got %d embeds", len(m.Attachments))
	}
	if !m.Attachments[0].isInline() || m.Attachments[0].ContentID != "logo" {
		t.Errorf("EmbedFile not inline: %+v", m.Attachments[0])
	}
	if m.Attachments[0].ContentType != "image/png" {
		t.Errorf("png type = %q", m.Attachments[0].ContentType)
	}
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("multipart/related")) {
		t.Error("expected multipart/related for embeds")
	}
}

func TestAttachURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		_, _ = w.Write([]byte("a,b,c\n1,2,3\n"))
	}))
	defer srv.Close()

	m := newFixed().SetText("report").AttachURL(srv.URL + "/data/report.csv")
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
	a := m.Attachments[0]
	if a.Filename != "report.csv" {
		t.Errorf("filename = %q", a.Filename)
	}
	if a.ContentType != "text/csv" {
		t.Errorf("content type = %q", a.ContentType)
	}
	if string(a.Content) != "a,b,c\n1,2,3\n" {
		t.Errorf("content = %q", a.Content)
	}
}

func TestAttachURLError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()
	m := newFixed().SetText("x").AttachURL(srv.URL + "/missing")
	if m.Err() == nil {
		t.Error("expected error for 404")
	}
}

func TestURLFilename(t *testing.T) {
	cases := map[string]string{
		"https://x.com/a/b/file.pdf": "file.pdf",
		"https://x.com/file.png?v=2": "file.png",
		"https://x.com/dir/":         "dir",
		"https://x.com":              "x.com",
		"https://x.com/p#frag":       "p",
	}
	for in, want := range cases {
		if got := urlFilename(in); got != want {
			t.Errorf("urlFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSniffContentType(t *testing.T) {
	// Explicit wins.
	if got := sniffContentType("application/json", "x.txt", nil); got != "application/json" {
		t.Errorf("explicit = %q", got)
	}
	// Extension used when explicit is empty/octet-stream.
	if got := sniffContentType("application/octet-stream", "page.html", nil); !strings.HasPrefix(got, "text/html") {
		t.Errorf("by-ext = %q", got)
	}
	// Content sniffing fallback.
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if got := sniffContentType("", "noext", png); got != "image/png" {
		t.Errorf("sniffed = %q", got)
	}
	// Ultimate fallback.
	if got := sniffContentType("", "noext", nil); got != "application/octet-stream" {
		t.Errorf("fallback = %q", got)
	}
}

func TestAttachURLDataRoundTrip(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), 200)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()
	m := newFixed().SetText("t").AttachURL(srv.URL + "/blob.bin")
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	part := extractPartBody(t, raw, "blob.bin")
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(part, "\r\n", ""))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, payload) {
		t.Error("round-trip mismatch")
	}
}
