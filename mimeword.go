package nodemailer

import (
	"crypto/rand"
	"encoding/base64"
	"mime"
	"strings"
)

// EncodeWord encodes s as an RFC 2047 "encoded-word" using the given charset
// (e.g. "utf-8") and Q (quoted-printable) encoding, as used for non-ASCII
// header values such as Subject and display names. ASCII-only input is returned
// unchanged. This mirrors nodemailer's libmime mimeWordEncode.
func EncodeWord(charset, s string) string {
	if isASCII(s) {
		return s
	}
	if charset == "" {
		charset = "utf-8"
	}
	return mime.QEncoding.Encode(charset, s)
}

// DecodeHeaderWord decodes a header value that may contain one or more RFC 2047
// encoded-words (either Q or B encoded, in any charset the standard library can
// map to UTF-8), returning the decoded UTF-8 text. Values without encoded-words
// are returned unchanged. This mirrors nodemailer's libmime mimeWordsDecode.
func DecodeHeaderWord(s string) (string, error) {
	dec := &mime.WordDecoder{}
	return dec.DecodeHeader(s)
}

// GenerateMessageID returns a fresh, globally-unique Message-ID value (without
// the surrounding angle brackets) of the form "<random>@domain". When domain is
// empty, "localhost" is used. The random component is cryptographically random,
// so the result is unique but not deterministic; set Message.MessageID
// explicitly for reproducible output.
func GenerateMessageID(domain string) string {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		domain = "localhost"
	}
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	return base64.RawURLEncoding.EncodeToString(buf[:]) + "@" + domain
}
