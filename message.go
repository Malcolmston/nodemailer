package nodemailer

import (
	"time"
)

// Attachment represents a file or inline resource attached to a Message.
type Attachment struct {
	// Filename is the name presented to the recipient.
	Filename string
	// Content holds the raw bytes of the attachment.
	Content []byte
	// ContentType is the MIME type, e.g. "application/pdf". If empty it is
	// guessed from the filename extension, falling back to
	// "application/octet-stream".
	ContentType string
	// ContentID, when set, adds a Content-ID header so the resource can be
	// referenced from HTML via cid:ContentID. Implies an inline part.
	ContentID string
	// Inline marks the attachment as an inline part (Content-Disposition:
	// inline) rather than a regular attachment. Setting ContentID also makes
	// the part inline.
	Inline bool
}

// isInline reports whether the attachment should be treated as inline.
func (a Attachment) isInline() bool {
	return a.Inline || a.ContentID != ""
}

// Message is a builder for an RFC 5322 / MIME email message.
//
// Fields may be set directly or via the fluent setter methods. Address setters
// parse and validate their inputs, deferring any error until Build is called.
type Message struct {
	From    Address
	To      []Address
	Cc      []Address
	Bcc     []Address
	ReplyTo []Address

	Subject string
	Text    string
	HTML    string

	// Headers holds additional custom headers in insertion order.
	Headers []Header

	// Date is the message Date header. If zero, the current time is used at
	// Build time; set it explicitly for deterministic output.
	Date time.Time

	// MessageID is the Message-ID header without angle brackets. If empty one
	// is generated at Build time; set it explicitly for deterministic output.
	MessageID string

	// Boundary is the base MIME multipart boundary. If empty a random one is
	// generated at Build time; set it explicitly for deterministic output.
	// Nested multiparts derive their boundaries from this base.
	Boundary string

	Attachments []Attachment

	// err holds the first error encountered by a fluent setter.
	err error
}

// Header is a single custom header field in insertion order.
type Header struct {
	Key   string
	Value string
}

// New returns an empty Message ready to be populated with the fluent setters.
func New() *Message {
	return &Message{}
}

// setErr records the first error seen by a fluent setter.
func (m *Message) setErr(err error) {
	if m.err == nil && err != nil {
		m.err = err
	}
}

// SetFrom parses and sets the From address.
func (m *Message) SetFrom(addr string) *Message {
	a, err := ParseAddress(addr)
	if err != nil {
		m.setErr(err)
		return m
	}
	m.From = a
	return m
}

// AddTo parses and appends one or more To recipients.
func (m *Message) AddTo(addrs ...string) *Message {
	return m.appendAddrs(&m.To, addrs)
}

// AddCc parses and appends one or more Cc recipients.
func (m *Message) AddCc(addrs ...string) *Message {
	return m.appendAddrs(&m.Cc, addrs)
}

// AddBcc parses and appends one or more Bcc recipients.
func (m *Message) AddBcc(addrs ...string) *Message {
	return m.appendAddrs(&m.Bcc, addrs)
}

// AddReplyTo parses and appends one or more Reply-To addresses.
func (m *Message) AddReplyTo(addrs ...string) *Message {
	return m.appendAddrs(&m.ReplyTo, addrs)
}

func (m *Message) appendAddrs(dst *[]Address, addrs []string) *Message {
	for _, s := range addrs {
		list, err := ParseAddressList(s)
		if err != nil {
			m.setErr(err)
			return m
		}
		*dst = append(*dst, list...)
	}
	return m
}

// SetSubject sets the Subject header.
func (m *Message) SetSubject(s string) *Message { m.Subject = s; return m }

// SetText sets the plain-text body.
func (m *Message) SetText(s string) *Message { m.Text = s; return m }

// SetHTML sets the HTML body.
func (m *Message) SetHTML(s string) *Message { m.HTML = s; return m }

// SetDate sets the Date header.
func (m *Message) SetDate(t time.Time) *Message { m.Date = t; return m }

// SetMessageID sets the Message-ID (without angle brackets).
func (m *Message) SetMessageID(id string) *Message { m.MessageID = id; return m }

// SetBoundary sets the base multipart boundary.
func (m *Message) SetBoundary(b string) *Message { m.Boundary = b; return m }

// AddHeader appends a custom header field.
func (m *Message) AddHeader(key, value string) *Message {
	m.Headers = append(m.Headers, Header{Key: key, Value: value})
	return m
}

// Attach appends an attachment.
func (m *Message) Attach(a Attachment) *Message {
	m.Attachments = append(m.Attachments, a)
	return m
}

// AttachBytes is a convenience for attaching in-memory content.
func (m *Message) AttachBytes(filename, contentType string, content []byte) *Message {
	return m.Attach(Attachment{Filename: filename, ContentType: contentType, Content: content})
}

// Embed appends an inline resource referenceable from HTML via cid:contentID.
func (m *Message) Embed(contentID, filename, contentType string, content []byte) *Message {
	return m.Attach(Attachment{
		Filename:    filename,
		ContentType: contentType,
		Content:     content,
		ContentID:   contentID,
	})
}

// Err returns the first error recorded by a fluent setter, if any.
func (m *Message) Err() error { return m.err }

// Recipients returns all envelope recipients (To, Cc and Bcc) as addr-specs.
func (m *Message) Recipients() []string {
	var out []string
	for _, group := range [][]Address{m.To, m.Cc, m.Bcc} {
		for _, a := range group {
			out = append(out, a.Address)
		}
	}
	return out
}
