package nodemailer

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestAlternativeBodies(t *testing.T) {
	m := newFixed().SetSubject("multi").
		SetText("plain").
		SetHTML("<p>html</p>").
		AddAlternative("text/x-web-markdown", "# markdown")
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "Content-Type: multipart/alternative;") {
		t.Error("expected multipart/alternative")
	}
	if !strings.Contains(s, "Content-Type: text/x-web-markdown; charset=utf-8") {
		t.Errorf("markdown alternative missing charset/type:\n%s", s)
	}
	// All three parts present in order.
	iText := strings.Index(s, "text/plain")
	iHTML := strings.Index(s, "text/html")
	iMD := strings.Index(s, "text/x-web-markdown")
	if iText >= iHTML || iHTML >= iMD {
		t.Errorf("alternative ordering wrong: text=%d html=%d md=%d", iText, iHTML, iMD)
	}
}

func TestICalEvent(t *testing.T) {
	ics := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nSUMMARY:Sync\r\nEND:VEVENT\r\nEND:VCALENDAR"
	m := newFixed().SetSubject("Invite").SetHTML("<p>See invite</p>")
	m.ICalEvent = &ICalEvent{Method: "request", Content: ics}
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "Content-Type: text/calendar; charset=utf-8; method=REQUEST") {
		t.Errorf("calendar part missing/wrong:\n%s", s)
	}
	// The base64 body must decode back to the original iCalendar text.
	idx := strings.Index(s, "method=REQUEST")
	rest := s[idx:]
	blank := strings.Index(rest, "\r\n\r\n")
	enc := rest[blank+4:]
	if end := strings.Index(enc, "\r\n--"); end >= 0 {
		enc = enc[:end]
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(enc, "\r\n", ""))
	if err != nil {
		t.Fatalf("decode calendar: %v", err)
	}
	if string(decoded) != ics {
		t.Errorf("calendar body mismatch:\n got %q\nwant %q", decoded, ics)
	}
}

func TestICalDefaultMethod(t *testing.T) {
	e := &ICalEvent{Content: "x"}
	if e.method() != "PUBLISH" {
		t.Errorf("default method = %q", e.method())
	}
}

func TestICalOnlyBody(t *testing.T) {
	// A message with only a calendar event (no text/html) still has content.
	m := New().SetFrom("a@b.com").AddTo("c@d.com")
	m.ICalEvent = &ICalEvent{Content: "BEGIN:VCALENDAR\r\nEND:VCALENDAR"}
	if _, err := m.Build(); err != nil {
		t.Fatalf("ical-only message failed to build: %v", err)
	}
}
