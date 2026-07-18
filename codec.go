package nodemailer

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// qpAllowed reports whether a byte may appear literally in Quoted-Printable
// output, per RFC 2045 section 6.7: TAB, CR, LF, and the printable ASCII ranges
// 0x20-0x3C and 0x3E-0x7E ("=", 0x3D, is always encoded).
func qpAllowed(b byte) bool {
	switch {
	case b == 0x09 || b == 0x0a || b == 0x0d:
		return true
	case b >= 0x20 && b <= 0x3c:
		return true
	case b >= 0x3e && b <= 0x7e:
		return true
	}
	return false
}

// QPEncode encodes data as a Quoted-Printable string with no line wrapping,
// mirroring nodemailer's libqp encode. Bytes outside the printable range are
// written as "=XX" (uppercase hex); a space or tab at the end of the input or
// immediately before a CR/LF is encoded so trailing whitespace is preserved.
// Use QPWrap to add soft line breaks.
func QPEncode(data []byte) string {
	var b strings.Builder
	n := len(data)
	for i := 0; i < n; i++ {
		c := data[i]
		if qpAllowed(c) && !((c == 0x20 || c == 0x09) && (i == n-1 || data[i+1] == 0x0a || data[i+1] == 0x0d)) {
			b.WriteByte(c)
			continue
		}
		fmt.Fprintf(&b, "=%02X", c)
	}
	return b.String()
}

var (
	reQPPartial2 = regexp.MustCompile(`(?i)[=][\da-f]{0,2}$`)
	reQPPartial1 = regexp.MustCompile(`(?i)[=][\da-f]{0,1}$`)
	reQPFull     = regexp.MustCompile(`(?i)^(?:=[\da-f]{2}){1,4}$`)
	reQPEnd2     = regexp.MustCompile(`(?i)[=][\da-f]{2}$`)
	reQPSpace    = regexp.MustCompile(`[ \t.,!?][^ \t.,!?]*$`)
)

// QPWrap inserts soft line breaks ("=\r\n") into an already Quoted-Printable
// encoded string so that no line exceeds lineLength characters, matching
// nodemailer's libqp wrap. A lineLength of zero (or less) defaults to 76.
// Breaks are chosen to avoid splitting "=XX" escape sequences and multi-byte
// UTF-8 runs, preferring word boundaries where possible.
func QPWrap(str string, lineLength int) string {
	if lineLength <= 0 {
		lineLength = 76
	}
	if len(str) <= lineLength {
		return str
	}

	pos := 0
	total := len(str)
	lineMargin := lineLength / 3
	var result strings.Builder

	for pos < total {
		end := pos + lineLength
		if end > total {
			end = total
		}
		line := str[pos:end]

		if idx := strings.Index(line, "\r\n"); idx >= 0 {
			line = line[:idx+2]
			result.WriteString(line)
			pos += len(line)
			continue
		}
		if strings.HasSuffix(line, "\n") {
			result.WriteString(line)
			pos += len(line)
			continue
		}

		tail := line
		if len(line) > lineMargin {
			tail = line[len(line)-lineMargin:]
		}
		if idx := strings.IndexByte(tail, '\n'); idx >= 0 {
			matchLen := len(tail) - idx
			line = line[:len(line)-(matchLen-1)]
			result.WriteString(line)
			pos += len(line)
			continue
		}

		if len(line) > lineLength-lineMargin {
			t2 := line
			if len(line) > lineMargin {
				t2 = line[len(line)-lineMargin:]
			}
			if loc := reQPSpace.FindStringIndex(t2); loc != nil {
				matchLen := len(t2) - loc[0]
				line = line[:len(line)-(matchLen-1)]
			} else if reQPPartial2.MatchString(line) {
				line = trimIncompleteQP(line, total, pos)
			}
		} else if reQPPartial2.MatchString(line) {
			line = trimIncompleteQP(line, total, pos)
		}

		if pos+len(line) < total && !strings.HasSuffix(line, "\n") {
			if len(line) == lineLength && reQPEnd2.MatchString(line) {
				line = line[:len(line)-3]
			} else if len(line) == lineLength {
				line = line[:len(line)-1]
			}
			pos += len(line)
			line += "=\r\n"
		} else {
			pos += len(line)
		}
		result.WriteString(line)
	}
	return result.String()
}

// trimIncompleteQP pulls back a line so it does not end mid-escape and does not
// split a multi-byte UTF-8 sequence, matching the tail handling in libqp wrap.
func trimIncompleteQP(line string, total, pos int) string {
	if m := reQPPartial1.FindString(line); m != "" {
		line = line[:len(line)-len(m)]
	}
	for len(line) > 3 && len(line) < total-pos && !reQPFull.MatchString(line) {
		m := reQPEnd2.FindString(line)
		if m == "" {
			break
		}
		code, err := strconv.ParseInt(m[1:3], 16, 32)
		if err != nil {
			break
		}
		if code < 128 {
			break
		}
		line = line[:len(line)-3]
		if code >= 0xc0 {
			break
		}
	}
	return line
}

// Base64Encode encodes data as standard (padded) base64 with no line wrapping,
// mirroring nodemailer's libbase64 encode. Use Base64Wrap to add line breaks.
func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64Wrap inserts CRLF line breaks into a base64 string so that no line
// exceeds lineLength characters, matching nodemailer's libbase64 wrap. A
// lineLength of zero (or less) defaults to 76. No trailing CRLF is emitted,
// even when the content is an exact multiple of lineLength.
func Base64Wrap(str string, lineLength int) string {
	if lineLength <= 0 {
		lineLength = 76
	}
	if len(str) <= lineLength {
		return str
	}
	chunkLength := lineLength * 1024
	var parts []string
	for pos := 0; pos < len(str); pos += chunkLength {
		end := pos + chunkLength
		if end > len(str) {
			end = len(str)
		}
		parts = append(parts, strings.TrimSpace(insertBase64Breaks(str[pos:end], lineLength)))
	}
	return strings.TrimSpace(strings.Join(parts, "\r\n"))
}

// insertBase64Breaks appends a CRLF after every full lineLength-character block.
func insertBase64Breaks(s string, n int) string {
	var b strings.Builder
	for len(s) >= n {
		b.WriteString(s[:n])
		b.WriteString("\r\n")
		s = s[n:]
	}
	b.WriteString(s)
	return b.String()
}

// IsPlainText reports whether value contains only characters that need no MIME
// encoding: printable ASCII plus TAB, CR and LF. It returns false for control
// characters (other than TAB/CR/LF) and any byte outside the 7-bit ASCII range.
// This mirrors nodemailer's mimeFuncs.isPlainText (non-parameter form).
func IsPlainText(value string) bool {
	for _, r := range value {
		if r >= 0x80 {
			return false
		}
		if r <= 0x08 || r == 0x0b || r == 0x0c || (r >= 0x0e && r <= 0x1f) {
			return false
		}
	}
	return true
}

// HasLongerLines reports whether str contains at least one line longer than
// lineLength characters, matching nodemailer's mimeFuncs.hasLongerLines. Lines
// are delimited by "\n" and measured in Unicode code points.
func HasLongerLines(str string, lineLength int) bool {
	for _, line := range strings.Split(str, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if utf8.RuneCountInString(line) > lineLength {
			return true
		}
	}
	return false
}

// qSafe reports whether an ASCII byte may appear literally in an RFC 2047
// "Q" encoded-word body (letters, digits and !*+-/=), per nodemailer's rule.
func qSafe(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z':
		return true
	case c >= 'A' && c <= 'Z':
		return true
	case c >= '0' && c <= '9':
		return true
	case c == '!' || c == '*' || c == '+' || c == '-' || c == '/' || c == '=':
		return true
	}
	return false
}

// EncodeMimeWord encodes data as a single RFC 2047 UTF-8 encoded-word using the
// given encoding ("Q" for quoted-printable, "B" for base64; empty defaults to
// "Q"), mirroring nodemailer's mimeFuncs.encodeWord without length splitting.
// The result is wrapped as "=?UTF-8?Q?...?=" or "=?UTF-8?B?...?=".
func EncodeMimeWord(data []byte, encoding string) string {
	enc := "Q"
	if e := strings.TrimSpace(strings.ToUpper(encoding)); e != "" {
		enc = e[:1]
	}
	if enc == "B" {
		return "=?UTF-8?B?" + Base64Encode(data) + "?="
	}
	qpStr := QPEncode(data)
	var b strings.Builder
	for i := 0; i < len(qpStr); i++ {
		c := qpStr[i]
		switch {
		case qSafe(c):
			b.WriteByte(c)
		case c == ' ':
			b.WriteByte('_')
		default:
			fmt.Fprintf(&b, "=%02X", c)
		}
	}
	return "=?UTF-8?Q?" + b.String() + "?="
}
