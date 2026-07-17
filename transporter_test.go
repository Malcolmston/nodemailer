package nodemailer

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestSendMailEndToEnd(t *testing.T) {
	mt := &MemoryTransport{}
	tr := NewTransporter(mt)

	msg := newFixed().
		AddCc("carl@example.com").
		AddBcc("secret@example.com").
		SetSubject("Hello").
		SetText("body")

	info, err := tr.SendMail(msg)
	if err != nil {
		t.Fatal(err)
	}
	if info.MessageID != "id@example.com" {
		t.Errorf("MessageID = %q", info.MessageID)
	}
	if info.Envelope.From != "ada@example.com" {
		t.Errorf("envelope From = %q", info.Envelope.From)
	}
	wantTo := []string{"grace@example.com", "carl@example.com", "secret@example.com"}
	if strings.Join(info.Envelope.To, ",") != strings.Join(wantTo, ",") {
		t.Errorf("envelope To = %v, want %v", info.Envelope.To, wantTo)
	}
	// Bcc must be an envelope recipient but must NOT appear in the headers.
	if bytes.Contains(info.Raw, []byte("secret@example.com")) {
		t.Error("Bcc leaked into message headers")
	}

	last, ok := mt.Last()
	if !ok || !bytes.Equal(last.Raw, info.Raw) {
		t.Error("transport did not receive the built message")
	}
}

// failTransport always fails, to exercise the error path.
type failTransport struct{}

func (failTransport) Send(string, []string, []byte) error {
	return errors.New("boom")
}

func TestSendMailBuildError(t *testing.T) {
	tr := NewTransporter(&MemoryTransport{})
	if _, err := tr.SendMail(New().SetFrom("bad")); err == nil {
		t.Error("expected build error to propagate")
	}
}

func TestSendMailTransportError(t *testing.T) {
	tr := NewTransporter(failTransport{})
	msg := newFixed().SetText("x")
	if _, err := tr.SendMail(msg); err == nil || err.Error() != "boom" {
		t.Errorf("expected transport error, got %v", err)
	}
}

func TestExtractMessageID(t *testing.T) {
	raw := []byte("From: <a@b.com>\r\nMessage-ID: <abc@host>\r\nSubject: x\r\n\r\nbody")
	if got := extractMessageID(raw); got != "abc@host" {
		t.Errorf("extractMessageID = %q", got)
	}
	if got := extractMessageID([]byte("From: <a@b.com>\r\n\r\nbody")); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestGeneratedMessageIDAndBoundary(t *testing.T) {
	// No fixed MessageID/Boundary: values are generated but well-formed.
	msg := New().SetFrom("ada@example.com").AddTo("grace@example.com").SetText("hi")
	raw, err := msg.Build()
	if err != nil {
		t.Fatal(err)
	}
	id := extractMessageID(raw)
	if !strings.HasSuffix(id, "@example.com") {
		t.Errorf("generated Message-ID = %q, want domain suffix", id)
	}
}
