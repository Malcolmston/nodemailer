package nodemailer

import (
	"bytes"
	"strings"
	"testing"
)

func TestStreamTransport(t *testing.T) {
	var buf bytes.Buffer
	tr := NewStreamTransport(&buf)

	raw, err := newFixed().SetSubject("Hi").SetText("Hello").Build()
	if err != nil {
		t.Fatal(err)
	}
	info, err := NewTransporter(tr).SendMail(newFixed().SetSubject("Hi").SetText("Hello"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), raw) {
		t.Errorf("stream output mismatch:\ngot:\n%s\nwant:\n%s", buf.Bytes(), raw)
	}
	if info.MessageID == "" {
		t.Error("expected a message id")
	}
}

func TestStreamTransportSeparatesMessages(t *testing.T) {
	var buf bytes.Buffer
	tr := NewStreamTransport(&buf)
	_ = tr.Send("a@x.com", []string{"b@y.com"}, []byte("MSG1"))
	_ = tr.Send("a@x.com", []string{"b@y.com"}, []byte("MSG2"))
	got := buf.String()
	if !strings.Contains(got, "MSG1") || !strings.Contains(got, "MSG2") {
		t.Fatalf("missing message bodies: %q", got)
	}
	if !strings.Contains(got, "MSG1\r\nMSG2") {
		t.Errorf("messages not CRLF-separated: %q", got)
	}
}
