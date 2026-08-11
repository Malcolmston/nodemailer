package nodemailer

import (
	"errors"
	"fmt"
	"strings"
)

// ErrHeaderInjection is returned by [Message.Build] when a header-valued field
// carries a bare carriage return or line feed. RFC 5322 forbids CR and LF
// inside a field body, and writing one verbatim ends the field early: every
// byte after it becomes a header of its own, or — after a blank line — a forged
// message body. That is email header injection (CWE-93).
//
// Non-ASCII text belongs in a header as an RFC 2047 encoded-word (which Build
// applies to the Subject automatically, and which [EncodeWord] exposes), never
// as raw bytes; there is therefore no legitimate value that needs a bare
// newline. Callers that must accept arbitrary untrusted text should fold it
// with [SanitizeHeaderValue] before handing it over.
var ErrHeaderInjection = errors.New("nodemailer: header injection: CR or LF in header value")

// SanitizeHeaderValue makes s safe to use as a header value by replacing every
// run of CR and LF bytes with a single space, so that an injected payload stays
// inside the field it was written to. It mirrors what nodemailer does with a
// newline-bearing header value and is the recommended way to pass text of
// unknown provenance (a ticket title, a user's display name, a search term) to
// [Message.SetSubject] or [Message.AddHeader] without risking
// [ErrHeaderInjection].
func SanitizeHeaderValue(s string) string {
	if !strings.ContainsAny(s, "\r\n") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	prevNewline := false
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '\r', '\n':
			if !prevNewline {
				b.WriteByte(' ')
			}
			prevNewline = true
		default:
			b.WriteByte(c)
			prevNewline = false
		}
	}
	return strings.TrimSpace(b.String())
}

// checkHeaderValue reports an [ErrHeaderInjection] error when value contains a
// bare CR or LF. field names the offending field for the error message.
func checkHeaderValue(field, value string) error {
	if strings.ContainsAny(value, "\r\n") {
		// %q escapes the CR/LF so the offending bytes are visible in the error.
		return fmt.Errorf("%w: %s: %q", ErrHeaderInjection, field, value)
	}
	return nil
}

// checkHeaderField validates a custom header field: the name must be a
// non-empty RFC 5322 field name (printable ASCII other than ':') and the value
// must be free of CR and LF.
func checkHeaderField(what string, h Header) error {
	if strings.TrimSpace(h.Key) == "" {
		return fmt.Errorf("nodemailer: %s: empty header name", what)
	}
	for i := 0; i < len(h.Key); i++ {
		if c := h.Key[i]; c <= ' ' || c >= 0x7f || c == ':' {
			return fmt.Errorf("%w: %s name %q", ErrHeaderInjection, what, h.Key)
		}
	}
	return checkHeaderValue(what+" "+h.Key, h.Value)
}

// checkAddresses validates the header-visible parts of an address list: the
// display name must not smuggle a newline into the To/Cc/Bcc/Reply-To field,
// and the addr-spec itself must remain well-formed.
func checkAddresses(field string, list []Address) error {
	for _, a := range list {
		if err := checkHeaderValue(field+" name", a.Name); err != nil {
			return err
		}
		if err := checkHeaderValue(field, a.Address); err != nil {
			return err
		}
	}
	return nil
}

// validateHeaders checks every field of m that reaches a header value. It is
// called by [Message.Build] before anything is encoded, so a newline-bearing
// value fails the build rather than reshaping the message.
//
// The check lives here, at the single encoding boundary, rather than in each
// fluent setter: the setters record values verbatim (Message fields are also
// exported and may be assigned directly), so validating at Build is what makes
// the guarantee complete.
func (m *Message) validateHeaders() error {
	if err := checkHeaderValue("Subject", m.Subject); err != nil {
		return err
	}
	if err := checkAddresses("From", []Address{m.From}); err != nil {
		return err
	}
	for _, l := range []struct {
		field string
		list  []Address
	}{
		{"To", m.To}, {"Cc", m.Cc}, {"Bcc", m.Bcc}, {"Reply-To", m.ReplyTo},
	} {
		if err := checkAddresses(l.field, l.list); err != nil {
			return err
		}
	}
	for _, g := range append(append([]AddressGroup{}, m.ToGroups...), m.CcGroups...) {
		if err := checkHeaderValue("address group name", g.Name); err != nil {
			return err
		}
		if err := checkAddresses("address group "+g.Name, g.Addresses); err != nil {
			return err
		}
	}
	if err := checkHeaderValue("Message-ID", m.MessageID); err != nil {
		return err
	}
	if err := checkHeaderValue("In-Reply-To", m.InReplyTo); err != nil {
		return err
	}
	for _, ref := range m.References {
		if err := checkHeaderValue("References", ref); err != nil {
			return err
		}
	}
	for _, h := range m.ListHeaders {
		if err := checkHeaderField("list header", h); err != nil {
			return err
		}
	}
	for _, h := range m.Headers {
		if err := checkHeaderField("header", h); err != nil {
			return err
		}
	}
	if err := checkHeaderValue("boundary", m.Boundary); err != nil {
		return err
	}
	for _, a := range m.Attachments {
		if err := checkHeaderValue("attachment filename", a.Filename); err != nil {
			return err
		}
		if err := checkHeaderValue("attachment content type", a.ContentType); err != nil {
			return err
		}
		if err := checkHeaderValue("attachment content ID", a.ContentID); err != nil {
			return err
		}
	}
	for _, alt := range m.Alternatives {
		if err := checkHeaderValue("alternative content type", alt.ContentType); err != nil {
			return err
		}
	}
	if m.ICalEvent != nil {
		if err := checkHeaderValue("calendar method", m.ICalEvent.Method); err != nil {
			return err
		}
		if err := checkHeaderValue("calendar filename", m.ICalEvent.Filename); err != nil {
			return err
		}
	}
	return nil
}
