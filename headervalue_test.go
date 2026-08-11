package nodemailer

import (
	"errors"
	"strings"
	"testing"
)

// base returns a minimal buildable message that the header-injection tests
// mutate one field at a time.
func base() *Message {
	return New().
		SetFrom("ada@example.com").
		AddTo("b@example.com").
		SetSubject("Innocent").
		SetText("body")
}

// TestSubjectCRLFIsRejected is the regression test for GHSA-pgqx-3gm6-xx62: a
// CRLF in the subject used to terminate the Subject field, turning the rest of
// the value into real headers (here a Bcc) and a forged body.
func TestSubjectCRLFIsRejected(t *testing.T) {
	m := base().SetSubject("Innocent\r\nBcc: attacker@example.com\r\n\r\ninjected body")
	out, err := m.Build()
	if !errors.Is(err, ErrHeaderInjection) {
		t.Fatalf("Build() error = %v, want ErrHeaderInjection", err)
	}
	if out != nil {
		t.Errorf("Build() returned %d bytes alongside the error", len(out))
	}
}

// TestHeaderValueCRLFIsRejected covers every header-valued field, not just the
// Subject: any one of them is an equally good injection sink.
func TestHeaderValueCRLFIsRejected(t *testing.T) {
	const payload = "value\r\nBcc: attacker@example.com"
	// From is validated a step earlier (Address.Validate), so it reports
	// ErrInvalidAddress; every other field reaches the Build-time header check.
	wantAddrErr := map[string]bool{"from name": true}
	cases := map[string]func(*Message){
		"subject":            func(m *Message) { m.Subject = payload },
		"custom header":      func(m *Message) { m.Headers = []Header{{"X-Ticket", payload}} },
		"custom header name": func(m *Message) { m.Headers = []Header{{"X-Bad\r\nBcc", "v"}} },
		"list header":        func(m *Message) { m.ListHeaders = []Header{{"List-ID", payload}} },
		"from name":          func(m *Message) { m.From = Address{Name: "Ada\r\nBcc: a@b.com", Address: "ada@example.com"} },
		"to name":            func(m *Message) { m.To = []Address{{Name: payload, Address: "b@example.com"}} },
		"cc address":         func(m *Message) { m.Cc = []Address{{Address: "c@example.com\r\nBcc: a@b.com"}} },
		"bcc name":           func(m *Message) { m.Bcc = []Address{{Name: payload, Address: "d@example.com"}} },
		"reply-to name":      func(m *Message) { m.ReplyTo = []Address{{Name: payload, Address: "e@example.com"}} },
		"group name":         func(m *Message) { m.ToGroups = []AddressGroup{{Name: payload}} },
		"group member": func(m *Message) {
			m.ToGroups = []AddressGroup{{Name: "Team", Addresses: []Address{{Name: payload, Address: "f@example.com"}}}}
		},
		"message-id":           func(m *Message) { m.MessageID = "id@example.com>\r\nBcc: a@b.com" },
		"in-reply-to":          func(m *Message) { m.InReplyTo = "id@example.com>\r\nBcc: a@b.com" },
		"references":           func(m *Message) { m.References = []string{"id@example.com>\r\nBcc: a@b.com"} },
		"boundary":             func(m *Message) { m.Boundary = "bnd\r\nBcc: a@b.com" },
		"attachment filename":  func(m *Message) { m.AttachBytes("a\r\nBcc: a@b.com.txt", "text/plain", []byte("x")) },
		"attachment mime type": func(m *Message) { m.AttachBytes("a.txt", "text/plain\r\nBcc: a@b.com", []byte("x")) },
		"attachment cid":       func(m *Message) { m.Embed("cid\r\nBcc: a@b.com", "a.png", "image/png", []byte("x")) },
		"alternative type":     func(m *Message) { m.AddAlternative("text/x-md\r\nBcc: a@b.com", "x") },
		"ical method": func(m *Message) {
			m.ICalEvent = &ICalEvent{Method: "REQUEST\r\nBcc: a@b.com", Content: "BEGIN:VCALENDAR"}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			m := base()
			mutate(m)
			out, err := m.Build()
			want, wantName := ErrHeaderInjection, "ErrHeaderInjection"
			if wantAddrErr[name] {
				want, wantName = ErrInvalidAddress, "ErrInvalidAddress"
			}
			if !errors.Is(err, want) {
				t.Fatalf("Build() error = %v, want %s\nmessage:\n%s", err, wantName, out)
			}
			if strings.Contains(string(out), "attacker@example.com") ||
				strings.Contains(string(out), "a@b.com") {
				t.Errorf("injected header reached the output:\n%s", out)
			}
		})
	}
}

// TestParseAddressRejectsNewlineInDisplayName checks the earlier of the two
// gates: a display name assigned through the validating constructors is
// refused before Build is ever reached.
func TestParseAddressRejectsNewlineInDisplayName(t *testing.T) {
	if err := (Address{Name: "Ada\r\nBcc: a@b.com", Address: "ada@example.com"}).Validate(); !errors.Is(err, ErrInvalidAddress) {
		t.Errorf("Validate() = %v, want ErrInvalidAddress", err)
	}
}

// TestLegitimateHeadersStillBuild is the positive half: ordinary values,
// including non-ASCII text carried as an RFC 2047 encoded-word and a folded
// long subject, must keep working unchanged.
func TestLegitimateHeadersStillBuild(t *testing.T) {
	m := New().
		SetFrom("Ada Lovelace <ada@example.com>").
		AddTo("Grüße <grace@example.com>").
		SetSubject("Grüß dich — Deploy done 🚀").
		AddHeader("X-Ticket", "ACME-1234").
		AddListHeader("ID", "announce.example.com").
		SetText("body").
		SetBoundary("BOUND").
		SetMessageID("fixed@example.com").
		AttachBytes("rapport-été.txt", "text/plain", []byte("hi"))
	out, err := m.Build()
	if err != nil {
		t.Fatalf("Build() = %v, want nil", err)
	}
	raw := string(out)
	// Non-ASCII must travel as an encoded-word, never as raw bytes.
	if !strings.Contains(raw, "Subject: =?utf-8?q?") {
		t.Errorf("subject not RFC 2047 encoded:\n%s", raw)
	}
	if strings.Contains(raw, "Grüß dich") {
		t.Errorf("raw non-ASCII bytes in header:\n%s", raw)
	}
	for _, want := range []string{"X-Ticket: ACME-1234", "List-ID: announce.example.com"} {
		if !strings.Contains(raw, want) {
			t.Errorf("missing %q in:\n%s", want, raw)
		}
	}
	// A long subject is still folded, and folding is the only place a CRLF may
	// appear in a header: it is always followed by whitespace.
	long := base().SetSubject(strings.Repeat("word ", 40))
	folded, err := long.Build()
	if err != nil {
		t.Fatalf("folded subject Build() = %v", err)
	}
	head, _, _ := strings.Cut(string(folded), "\r\n\r\n")
	for _, line := range strings.Split(head, "\r\n")[1:] {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		if !strings.Contains(line, ":") {
			t.Errorf("header block line is neither a field nor a continuation: %q", line)
		}
	}
}

// TestSanitizeHeaderValue documents the escape hatch offered to callers that
// must accept arbitrary text: fold the newlines instead of failing the build.
func TestSanitizeHeaderValue(t *testing.T) {
	tests := []struct{ in, want string }{
		{"plain", "plain"},
		{"Innocent\r\nBcc: attacker@example.com", "Innocent Bcc: attacker@example.com"},
		{"a\n\n\nb", "a b"},
		{"\r\nlead and trail\r\n", "lead and trail"},
	}
	for _, tt := range tests {
		if got := SanitizeHeaderValue(tt.in); got != tt.want {
			t.Errorf("SanitizeHeaderValue(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
	m := base().SetSubject(SanitizeHeaderValue("Innocent\r\nBcc: attacker@example.com"))
	out, err := m.Build()
	if err != nil {
		t.Fatalf("Build() after sanitising = %v", err)
	}
	if strings.Contains(string(out), "\r\nBcc:") {
		t.Errorf("sanitised subject still produced a Bcc header:\n%s", out)
	}
}
