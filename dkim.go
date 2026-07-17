package nodemailer

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Canonicalization selects a DKIM canonicalization algorithm (RFC 6376 §3.4).
type Canonicalization string

const (
	// CanonSimple is the "simple" canonicalization: header names and values are
	// used verbatim (only trailing empty body lines are stripped).
	CanonSimple Canonicalization = "simple"
	// CanonRelaxed is the "relaxed" canonicalization: header names are
	// lower-cased, whitespace runs collapsed, and body lines stripped of
	// trailing whitespace.
	CanonRelaxed Canonicalization = "relaxed"
)

// DKIM holds the configuration for signing a message with a DKIM-Signature
// header (RFC 6376) using RSA-SHA256.
type DKIM struct {
	// Domain is the signing domain (the "d=" tag), e.g. "example.com".
	Domain string
	// Selector is the key selector (the "s=" tag), e.g. "mail". The public key
	// is published at <selector>._domainkey.<domain> in DNS.
	Selector string
	// PrivateKey is the RSA private key used to sign.
	PrivateKey *rsa.PrivateKey
	// Headers lists the header field names to sign (the "h=" tag), in order. If
	// empty, DefaultDKIMHeaders is used. Missing headers are skipped.
	Headers []string
	// HeaderCanon is the header canonicalization; defaults to CanonRelaxed.
	HeaderCanon Canonicalization
	// BodyCanon is the body canonicalization; defaults to CanonRelaxed.
	BodyCanon Canonicalization
	// Time is the signature timestamp (the "t=" tag). If zero, no timestamp is
	// added; set it explicitly for deterministic output.
	Time time.Time
}

// DefaultDKIMHeaders is the default set of headers to sign when DKIM.Headers is
// empty. Only headers present in the message are included.
var DefaultDKIMHeaders = []string{"From", "To", "Cc", "Subject", "Date", "Message-ID", "MIME-Version", "Content-Type"}

// ErrDKIMConfig is returned when the DKIM configuration is incomplete.
var ErrDKIMConfig = errors.New("nodemailer: incomplete DKIM configuration")

// ParseRSAPrivateKey decodes a PEM-encoded RSA private key in either PKCS#1
// ("RSA PRIVATE KEY") or PKCS#8 ("PRIVATE KEY") form.
func ParseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("nodemailer: no PEM block found in key")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("nodemailer: parse private key: %w", err)
	}
	rsaKey, ok := keyAny.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("nodemailer: PEM key is not an RSA key")
	}
	return rsaKey, nil
}

// Sign returns a DKIM-Signature header line (without a trailing CRLF) for the
// given fully-encoded message. The message must have its header/body separator
// as an empty CRLF line. The returned line should be prepended to the message
// headers.
func (d *DKIM) Sign(raw []byte) (string, error) {
	if d.Domain == "" || d.Selector == "" || d.PrivateKey == nil {
		return "", ErrDKIMConfig
	}
	headerCanon := d.HeaderCanon
	if headerCanon == "" {
		headerCanon = CanonRelaxed
	}
	bodyCanon := d.BodyCanon
	if bodyCanon == "" {
		bodyCanon = CanonRelaxed
	}

	headerBlock, body := splitMessage(raw)
	headers := parseHeaderFields(headerBlock)

	signedNames := d.Headers
	if len(signedNames) == 0 {
		signedNames = DefaultDKIMHeaders
	}

	// Select, in order, the header fields that are present in the message.
	var selected []headerField
	var hTag []string
	for _, name := range signedNames {
		if hf, ok := findLastHeader(headers, name); ok {
			selected = append(selected, hf)
			hTag = append(hTag, hf.name)
		}
	}

	// Body hash.
	canonBody := canonicalizeBody(body, bodyCanon)
	bodyHash := sha256.Sum256(canonBody)
	bh := base64.StdEncoding.EncodeToString(bodyHash[:])

	// Build the DKIM-Signature value with an empty b= tag for signing.
	var sb strings.Builder
	sb.WriteString("v=1; a=rsa-sha256; c=")
	sb.WriteString(string(headerCanon))
	sb.WriteString("/")
	sb.WriteString(string(bodyCanon))
	sb.WriteString("; d=")
	sb.WriteString(d.Domain)
	sb.WriteString("; s=")
	sb.WriteString(d.Selector)
	if !d.Time.IsZero() {
		sb.WriteString("; t=")
		fmt.Fprintf(&sb, "%d", d.Time.Unix())
	}
	sb.WriteString("; bh=")
	sb.WriteString(bh)
	sb.WriteString("; h=")
	sb.WriteString(strings.Join(hTag, ":"))
	sb.WriteString("; b=")
	sigValue := sb.String()

	// Canonicalize the signed headers plus the DKIM-Signature header itself
	// (with empty b=), per RFC 6376 §3.7.
	var canonHeaders bytes.Buffer
	for _, hf := range selected {
		canonHeaders.WriteString(canonicalizeHeader(hf, headerCanon))
		canonHeaders.WriteString(crlf)
	}
	// The DKIM-Signature header is canonicalized with no trailing CRLF.
	dkimField := headerField{name: "DKIM-Signature", value: " " + sigValue}
	canonHeaders.WriteString(canonicalizeHeader(dkimField, headerCanon))

	hashed := sha256.Sum256(canonHeaders.Bytes())
	sig, err := rsa.SignPKCS1v15(rand.Reader, d.PrivateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", fmt.Errorf("nodemailer: DKIM sign: %w", err)
	}
	b := base64.StdEncoding.EncodeToString(sig)

	return "DKIM-Signature: " + sigValue + b, nil
}

// headerField is a single unfolded header field as it appeared in the message.
type headerField struct {
	name  string // canonical name as written, e.g. "Subject"
	value string // everything after the colon, including leading space and folds
}

// splitMessage splits raw into the header block (without the terminating blank
// line) and the body.
func splitMessage(raw []byte) (header, body []byte) {
	sep := []byte(crlf + crlf)
	if i := bytes.Index(raw, sep); i >= 0 {
		return raw[:i], raw[i+len(sep):]
	}
	return raw, nil
}

// parseHeaderFields parses a header block into ordered header fields, joining
// folded continuation lines.
func parseHeaderFields(block []byte) []headerField {
	lines := strings.Split(string(block), crlf)
	var fields []headerField
	for _, line := range lines {
		if line == "" {
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			// Continuation of the previous field.
			if len(fields) > 0 {
				fields[len(fields)-1].value += crlf + line
			}
			continue
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		fields = append(fields, headerField{
			name:  line[:colon],
			value: line[colon+1:],
		})
	}
	return fields
}

// findLastHeader returns the last header field with the given name
// (case-insensitive), matching RFC 6376's bottom-up header selection.
func findLastHeader(fields []headerField, name string) (headerField, bool) {
	for i := len(fields) - 1; i >= 0; i-- {
		if strings.EqualFold(fields[i].name, name) {
			return fields[i], true
		}
	}
	return headerField{}, false
}

// canonicalizeHeader canonicalizes a single header field per RFC 6376 §3.4.
func canonicalizeHeader(hf headerField, canon Canonicalization) string {
	if canon == CanonSimple {
		// Simple: the header is used verbatim, name + ":" + value.
		return hf.name + ":" + hf.value
	}
	// Relaxed: lower-case name, unfold, collapse internal whitespace runs to a
	// single space, strip leading/trailing whitespace of the value.
	name := strings.ToLower(strings.TrimSpace(hf.name))
	value := unfoldAndCollapse(hf.value)
	return name + ":" + value
}

// unfoldAndCollapse removes folding and collapses all whitespace runs (spaces,
// tabs, CRLF) into single spaces, trimming the result.
func unfoldAndCollapse(s string) string {
	var b strings.Builder
	inWS := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			inWS = true
			continue
		}
		if inWS && b.Len() > 0 {
			b.WriteByte(' ')
		}
		inWS = false
		b.WriteByte(c)
	}
	return b.String()
}

// canonicalizeBody canonicalizes the message body per RFC 6376 §3.4.3/§3.4.4.
func canonicalizeBody(body []byte, canon Canonicalization) []byte {
	if len(body) == 0 {
		// An empty body canonicalizes to a single CRLF.
		return []byte(crlf)
	}
	s := string(body)
	if canon == CanonRelaxed {
		lines := strings.Split(s, crlf)
		for i, ln := range lines {
			// Strip trailing whitespace and collapse internal WS runs.
			lines[i] = collapseTrailing(ln)
		}
		s = strings.Join(lines, crlf)
	}
	// Both algorithms: remove all trailing empty lines, then ensure exactly one
	// terminating CRLF.
	s = strings.TrimRight(s, "\r\n")
	return []byte(s + crlf)
}

// collapseTrailing collapses internal whitespace runs to a single space and
// strips trailing whitespace from a single body line (relaxed body canon).
func collapseTrailing(line string) string {
	var b strings.Builder
	inWS := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == ' ' || c == '\t' {
			inWS = true
			continue
		}
		if inWS {
			b.WriteByte(' ')
			inWS = false
		}
		b.WriteByte(c)
	}
	return b.String()
}
