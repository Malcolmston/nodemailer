package nodemailer

import (
	"html"
	"regexp"
	"strings"
)

// htmlBlockTags are HTML elements whose boundaries should become line breaks
// when the markup is flattened to plain text.
var htmlBlockTags = map[string]bool{
	"p": true, "div": true, "br": true, "tr": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"table": true, "blockquote": true, "section": true, "article": true,
	"header": true, "footer": true, "ul": true, "ol": true, "pre": true,
}

var (
	htmlScriptStyleRe = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	htmlTagRe         = regexp.MustCompile(`(?s)<(/?)([a-zA-Z][a-zA-Z0-9]*)[^>]*>`)
	htmlWSRe          = regexp.MustCompile(`[ \t\f\v]+`)
	htmlBlankLinesRe  = regexp.MustCompile(`\n{3,}`)
)

// HTMLToText converts an HTML fragment or document into a readable plain-text
// approximation. Script and style contents are dropped, block-level tags become
// line breaks, list items are prefixed with a bullet, HTML entities are decoded
// and runs of whitespace are collapsed.
//
// It is a best-effort flattener intended for generating a text/plain
// alternative from an HTML body, not a full HTML renderer.
func HTMLToText(htmlSource string) string {
	s := htmlScriptStyleRe.ReplaceAllString(htmlSource, "")

	var b strings.Builder
	last := 0
	for _, m := range htmlTagRe.FindAllStringSubmatchIndex(s, -1) {
		b.WriteString(s[last:m[0]])
		last = m[1]
		closing := s[m[2]:m[3]] == "/"
		tag := strings.ToLower(s[m[4]:m[5]])
		switch {
		case tag == "br":
			b.WriteString("\n")
		case tag == "li" && !closing:
			b.WriteString("\n* ")
		case htmlBlockTags[tag]:
			b.WriteString("\n")
		}
	}
	b.WriteString(s[last:])

	text := html.UnescapeString(b.String())
	text = htmlWSRe.ReplaceAllString(text, " ")

	// Trim trailing/leading spaces around each line.
	lines := strings.Split(text, "\n")
	for i, ln := range lines {
		lines[i] = strings.TrimSpace(ln)
	}
	text = strings.Join(lines, "\n")

	text = htmlBlankLinesRe.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

// GenerateTextFromHTML populates the message Text body from its HTML body using
// HTMLToText, mirroring nodemailer's generateTextFromHtml option. It is a no-op
// when the HTML body is empty or a Text body is already set. It returns the
// message for chaining.
func (m *Message) GenerateTextFromHTML() *Message {
	if m.Text == "" && m.HTML != "" {
		m.Text = HTMLToText(m.HTML)
	}
	return m
}
