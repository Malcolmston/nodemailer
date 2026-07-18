package nodemailer

import (
	"bytes"
	"testing"
)

func TestParseMIMERoundTrip(t *testing.T) {
	raw, err := newFixed().
		SetSubject("Grüße from Ada").
		SetText("Plain body").
		SetHTML("<p>HTML body</p>").
		AttachBytes("report.txt", "text/plain", []byte("file contents")).
		Embed("logo", "logo.png", "image/png", []byte{0x89, 0x50, 0x4e, 0x47}).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	pm, err := ParseMIME(raw)
	if err != nil {
		t.Fatal(err)
	}

	if pm.Subject != "Grüße from Ada" {
		t.Errorf("Subject = %q", pm.Subject)
	}
	if len(pm.From) != 1 || pm.From[0].Address != "ada@example.com" {
		t.Errorf("From = %+v", pm.From)
	}
	if len(pm.To) != 1 || pm.To[0].Address != "grace@example.com" {
		t.Errorf("To = %+v", pm.To)
	}
	if pm.MessageID != "id@example.com" {
		t.Errorf("MessageID = %q", pm.MessageID)
	}
	if pm.Text != "Plain body" {
		t.Errorf("Text = %q", pm.Text)
	}
	if pm.HTML != "<p>HTML body</p>" {
		t.Errorf("HTML = %q", pm.HTML)
	}

	if len(pm.Attachments) != 2 {
		t.Fatalf("got %d attachments, want 2", len(pm.Attachments))
	}
	var gotFile, gotEmbed bool
	for _, a := range pm.Attachments {
		switch a.Filename {
		case "report.txt":
			gotFile = true
			if string(a.Content) != "file contents" {
				t.Errorf("attachment content = %q", a.Content)
			}
		case "logo.png":
			gotEmbed = true
			if a.ContentID != "logo" {
				t.Errorf("ContentID = %q", a.ContentID)
			}
			if !a.Inline {
				t.Error("embed should be inline")
			}
			if !bytes.Equal(a.Content, []byte{0x89, 0x50, 0x4e, 0x47}) {
				t.Errorf("embed content = %x", a.Content)
			}
		}
	}
	if !gotFile || !gotEmbed {
		t.Errorf("missing attachments: file=%v embed=%v", gotFile, gotEmbed)
	}
}

func TestParseMIMEGet(t *testing.T) {
	raw, err := newFixed().SetSubject("Grüße").SetText("hi").Build()
	if err != nil {
		t.Fatal(err)
	}
	pm, err := ParseMIME(raw)
	if err != nil {
		t.Fatal(err)
	}
	if pm.Get("Subject") != "Grüße" {
		t.Errorf("Get(Subject) = %q", pm.Get("Subject"))
	}
	if pm.Get("Nonexistent") != "" {
		t.Error("expected empty for missing header")
	}
}

func TestParseMIMEInvalid(t *testing.T) {
	if _, err := ParseMIME([]byte("this is not a message")); err == nil {
		t.Error("expected error for malformed input")
	}
}
