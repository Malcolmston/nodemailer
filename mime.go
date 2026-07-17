package nodemailer

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	stdpath "path"
	"strings"
	"time"
)

// crlf is the canonical MIME line ending.
const crlf = "\r\n"

// mimePart is a node in the MIME tree. A node is either a leaf (body set,
// children nil) or a multipart container (children set, subtype non-empty).
type mimePart struct {
	headers  []Header
	body     []byte
	subtype  string // e.g. "alternative", "mixed", "related"
	boundary string
	children []*mimePart
}

// Build encodes the Message into raw RFC 5322 / MIME bytes with CRLF line
// endings. Output is deterministic when Date, MessageID and Boundary are all
// set explicitly.
func (m *Message) Build() ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	if err := m.From.Validate(); err != nil {
		return nil, fmt.Errorf("nodemailer: From: %w", err)
	}
	if len(m.To) == 0 && len(m.Cc) == 0 && len(m.Bcc) == 0 &&
		len(m.ToGroups) == 0 && len(m.CcGroups) == 0 {
		return nil, errors.New("nodemailer: message has no recipients")
	}
	if m.Text == "" && m.HTML == "" && len(m.Attachments) == 0 &&
		len(m.Alternatives) == 0 && m.ICalEvent == nil {
		return nil, errors.New("nodemailer: message has no content")
	}

	base := m.Boundary
	if base == "" {
		b, err := randomBoundary()
		if err != nil {
			return nil, err
		}
		base = b
	}
	bs := &boundarySeq{base: base}

	root, err := m.buildRoot(bs)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	m.writeTopHeaders(&buf, root)
	buf.WriteString(crlf)
	renderPartBody(&buf, root)
	out := buf.Bytes()

	// DKIM signing: sign the finished message and prepend the DKIM-Signature
	// header so it precedes the fields it covers.
	if m.DKIM != nil {
		sig, err := m.DKIM.Sign(out)
		if err != nil {
			return nil, err
		}
		out = append([]byte(sig+crlf), out...)
	}
	return out, nil
}

// buildRoot assembles the MIME tree for the message body and attachments.
func (m *Message) buildRoot(bs *boundarySeq) (*mimePart, error) {
	body := m.buildBody(bs)

	var inline, regular []Attachment
	for _, a := range m.Attachments {
		if a.isInline() {
			inline = append(inline, a)
		} else {
			regular = append(regular, a)
		}
	}

	// Wrap body + inline resources in multipart/related.
	if len(inline) > 0 {
		related := &mimePart{
			subtype:  "related",
			boundary: bs.next(),
			children: []*mimePart{body},
		}
		for _, a := range inline {
			related.children = append(related.children, attachmentPart(a))
		}
		body = related
	}

	// Wrap in multipart/mixed when there are regular attachments.
	if len(regular) > 0 {
		mixed := &mimePart{
			subtype:  "mixed",
			boundary: bs.next(),
			children: []*mimePart{body},
		}
		for _, a := range regular {
			mixed.children = append(mixed.children, attachmentPart(a))
		}
		body = mixed
	}
	return body, nil
}

// buildBody builds the text/html portion: a single leaf, or a
// multipart/alternative when several body representations are present (plain
// text, HTML, additional alternatives and any calendar event).
func (m *Message) buildBody(bs *boundarySeq) *mimePart {
	var parts []*mimePart
	if m.Text != "" {
		parts = append(parts, textPart("text/plain", m.Text))
	}
	if m.HTML != "" {
		parts = append(parts, textPart("text/html", m.HTML))
	}
	for _, alt := range m.Alternatives {
		parts = append(parts, alt.part())
	}
	if m.ICalEvent != nil {
		parts = append(parts, m.ICalEvent.part())
	}
	switch len(parts) {
	case 0:
		// No body at all (attachments-only message): empty text part.
		return textPart("text/plain", "")
	case 1:
		return parts[0]
	default:
		return &mimePart{
			subtype:  "alternative",
			boundary: bs.next(),
			children: parts,
		}
	}
}

// textPart builds a quoted-printable-encoded leaf for a text body, appending a
// utf-8 charset parameter to the media type.
func textPart(mediaType, content string) *mimePart {
	return textPartCT(mediaType+"; charset=utf-8", content)
}

// textPartCT builds a quoted-printable-encoded leaf for a text body using the
// full content-type value as given, adding a utf-8 charset parameter when the
// type is textual and none is present.
func textPartCT(contentType, content string) *mimePart {
	if !strings.Contains(strings.ToLower(contentType), "charset") &&
		strings.HasPrefix(strings.ToLower(contentType), "text/") {
		contentType += "; charset=utf-8"
	}
	var qp bytes.Buffer
	w := quotedprintable.NewWriter(&qp)
	// Normalise to CRLF-free input; quotedprintable.Writer emits CRLF itself.
	_, _ = w.Write([]byte(strings.ReplaceAll(content, "\r\n", "\n")))
	_ = w.Close()
	return &mimePart{
		headers: []Header{
			{"Content-Type", contentType},
			{"Content-Transfer-Encoding", "quoted-printable"},
		},
		body: qp.Bytes(),
	}
}

// attachmentPart builds a base64-encoded leaf for an attachment.
func attachmentPart(a Attachment) *mimePart {
	ct := a.ContentType
	if ct == "" {
		ct = mime.TypeByExtension(stdpath.Ext(a.Filename))
	}
	if ct == "" {
		ct = "application/octet-stream"
	}

	disposition := "attachment"
	if a.isInline() {
		disposition = "inline"
	}

	headers := []Header{
		{"Content-Type", ctWithName(ct, a.Filename)},
		{"Content-Transfer-Encoding", "base64"},
		{"Content-Disposition", dispositionWithName(disposition, a.Filename)},
	}
	if a.ContentID != "" {
		headers = append(headers, Header{"Content-ID", "<" + a.ContentID + ">"})
	}

	return &mimePart{
		headers: headers,
		body:    base64Wrap(a.Content),
	}
}

// ctWithName appends a name parameter to a content type when a filename is set.
func ctWithName(ct, filename string) string {
	if filename == "" {
		return ct
	}
	return ct + "; name=" + quoteParam(filename)
}

// dispositionWithName builds a Content-Disposition value with a filename param.
func dispositionWithName(disposition, filename string) string {
	if filename == "" {
		return disposition
	}
	return disposition + "; filename=" + quoteParam(filename)
}

// quoteParam encodes a MIME parameter value, applying RFC 2047 for non-ASCII
// and quoting otherwise.
func quoteParam(s string) string {
	if isASCII(s) {
		return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
	}
	return mime.BEncoding.Encode("utf-8", s)
}

// writeTopHeaders writes the message-level headers followed by the root part's
// own content headers.
func (m *Message) writeTopHeaders(buf *bytes.Buffer, root *mimePart) {
	date := m.Date
	if date.IsZero() {
		date = time.Now()
	}
	msgID := m.MessageID
	if msgID == "" {
		msgID = generateMessageID(m.From.Address)
	}

	writeHeader(buf, "Date", date.Format(time.RFC1123Z))
	writeHeader(buf, "From", m.From.String())
	if v := recipientHeader(m.To, m.ToGroups); v != "" {
		writeHeader(buf, "To", v)
	}
	if v := recipientHeader(m.Cc, m.CcGroups); v != "" {
		writeHeader(buf, "Cc", v)
	}
	if len(m.ReplyTo) > 0 {
		writeHeader(buf, "Reply-To", addressListString(m.ReplyTo))
	}
	writeHeader(buf, "Message-ID", "<"+msgID+">")
	if m.InReplyTo != "" {
		writeHeader(buf, "In-Reply-To", "<"+strings.Trim(m.InReplyTo, "<>")+">")
	}
	if len(m.References) > 0 {
		writeHeader(buf, "References", m.referencesValue())
	}
	writeHeader(buf, "Subject", encodeHeaderWord(m.Subject))
	for _, h := range m.Priority.headers() {
		writeHeader(buf, h.Key, h.Value)
	}
	for _, h := range m.ListHeaders {
		writeHeader(buf, h.Key, h.Value)
	}
	for _, h := range m.Headers {
		writeHeader(buf, h.Key, h.Value)
	}
	writeHeader(buf, "MIME-Version", "1.0")
	// The root part's content headers belong on the top-level message.
	for _, h := range root.contentHeaders() {
		writeHeader(buf, h.Key, h.Value)
	}
}

// contentHeaders returns the headers describing a part's content, synthesising
// the Content-Type for multipart containers.
func (p *mimePart) contentHeaders() []Header {
	if p.subtype != "" {
		return []Header{
			{"Content-Type", fmt.Sprintf("multipart/%s; boundary=%q", p.subtype, p.boundary)},
		}
	}
	return p.headers
}

// renderPartBody writes the body (content after the blank line) of a part.
//
// For a multipart container each child is written as
//
//	--boundary CRLF headers CRLF CRLF body CRLF
//
// followed by the closing --boundary-- CRLF. The CRLF that terminates each
// body combines with the following "--boundary" to form the RFC 2046
// delimiter (CRLF dash-boundary). Leaf bodies therefore carry no trailing CRLF
// of their own.
func renderPartBody(buf *bytes.Buffer, p *mimePart) {
	if p.subtype == "" {
		buf.Write(p.body)
		return
	}
	for _, child := range p.children {
		buf.WriteString("--" + p.boundary + crlf)
		for _, h := range child.contentHeaders() {
			writeHeader(buf, h.Key, h.Value)
		}
		buf.WriteString(crlf)
		renderPartBody(buf, child)
		buf.WriteString(crlf)
	}
	buf.WriteString("--" + p.boundary + "--" + crlf)
}

// writeHeader writes a single folded header field terminated by CRLF.
func writeHeader(buf *bytes.Buffer, key, value string) {
	buf.WriteString(foldHeader(key, value))
	buf.WriteString(crlf)
}

// foldHeader folds a header line at whitespace to keep lines <= 78 chars,
// using CRLF + single-space continuation. Encoded words and quoted strings are
// treated as indivisible tokens because they never contain spaces.
func foldHeader(key, value string) string {
	const limit = 78
	prefix := key + ": "
	if len(prefix)+len(value) <= limit {
		return prefix + value
	}
	var b strings.Builder
	b.WriteString(prefix)
	lineLen := len(prefix)
	tokens := strings.Split(value, " ")
	for i, tok := range tokens {
		if i > 0 {
			if lineLen+1+len(tok) > limit {
				b.WriteString(crlf + " ")
				lineLen = 1
			} else {
				b.WriteString(" ")
				lineLen++
			}
		}
		b.WriteString(tok)
		lineLen += len(tok)
	}
	return b.String()
}

// encodeHeaderWord applies RFC 2047 encoded-word encoding when the value
// contains non-ASCII characters; otherwise it is returned unchanged.
func encodeHeaderWord(s string) string {
	if isASCII(s) {
		return s
	}
	return mime.QEncoding.Encode("utf-8", s)
}

// base64Wrap base64-encodes data and hard-wraps it at 76 characters per line
// with CRLF endings. There is no trailing CRLF after the final line; the
// enclosing multipart renderer supplies the delimiter's leading CRLF.
func base64Wrap(data []byte) []byte {
	encoded := base64.StdEncoding.EncodeToString(data)
	var b bytes.Buffer
	for len(encoded) > 76 {
		b.WriteString(encoded[:76])
		b.WriteString(crlf)
		encoded = encoded[76:]
	}
	b.WriteString(encoded)
	return b.Bytes()
}

// isASCII reports whether s contains only 7-bit ASCII bytes.
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// boundarySeq generates deterministic, distinct boundaries derived from a base.
type boundarySeq struct {
	base string
	n    int
}

func (b *boundarySeq) next() string {
	b.n++
	if b.n == 1 {
		return b.base
	}
	return fmt.Sprintf("%s_%d", b.base, b.n)
}

// randomBoundary produces a random MIME boundary.
func randomBoundary() (string, error) {
	var buf [18]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return "----=_NodemailerPart_" + base64.RawURLEncoding.EncodeToString(buf[:]), nil
}

// generateMessageID builds a Message-ID from the sender's domain and random
// data.
func generateMessageID(from string) string {
	domain := "localhost"
	if at := strings.LastIndex(from, "@"); at >= 0 && at < len(from)-1 {
		domain = from[at+1:]
	}
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	return base64.RawURLEncoding.EncodeToString(buf[:]) + "@" + domain
}
