package nodemailer

// Alternative is an additional body representation placed alongside the plain
// text and HTML parts inside the multipart/alternative container. Mail clients
// pick the last (most preferred) alternative they can render.
type Alternative struct {
	// ContentType is the MIME media type, e.g. "text/x-web-markdown". A charset
	// parameter of utf-8 is appended automatically for text/* types that lack
	// one.
	ContentType string
	// Content is the body text.
	Content string
}

// AddAlternative appends an alternative body representation with the given
// content type.
func (m *Message) AddAlternative(contentType, content string) *Message {
	m.Alternatives = append(m.Alternatives, Alternative{ContentType: contentType, Content: content})
	return m
}

// part builds the MIME leaf for an alternative body.
func (a Alternative) part() *mimePart {
	ct := a.ContentType
	if ct == "" {
		ct = "text/plain"
	}
	return textPartCT(ct, a.Content)
}
