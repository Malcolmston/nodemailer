package nodemailer

// SetAMP adds an AMP for Email body as a "text/x-amp-html" alternative,
// mirroring nodemailer's amp message option. Supporting clients render the AMP
// body in preference to the plain HTML one; others fall back to HTML or text.
// It returns the message for chaining.
func (m *Message) SetAMP(ampHTML string) *Message {
	return m.AddAlternative("text/x-amp-html", ampHTML)
}

// SetWatchHTML adds an Apple Watch specific body as a "text/watch-html"
// alternative, mirroring nodemailer's watchHtml message option. It returns the
// message for chaining.
func (m *Message) SetWatchHTML(watchHTML string) *Message {
	return m.AddAlternative("text/watch-html", watchHTML)
}
