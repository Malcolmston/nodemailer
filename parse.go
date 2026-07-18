package nodemailer

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
	"time"
)

// ParsedMessage is the decoded representation of a raw RFC 5322 / MIME message,
// providing read access to its headers, addresses, bodies and attachments. It
// is produced by ParseMIME and mirrors, in a lightweight form, the output of
// nodemailer's companion mailparser.
type ParsedMessage struct {
	// Header holds the top-level message headers.
	Header mail.Header
	// From, To and Cc are the parsed address lists (empty when the header is
	// absent or unparseable).
	From []Address
	To   []Address
	Cc   []Address
	// Subject is the RFC 2047 decoded Subject header.
	Subject string
	// Date is the parsed Date header, or the zero time when absent/invalid.
	Date time.Time
	// MessageID is the Message-ID value with any angle brackets removed.
	MessageID string
	// Text is the concatenated text/plain body content, if any.
	Text string
	// HTML is the concatenated text/html body content, if any.
	HTML string
	// Attachments holds every part with an attachment or inline disposition.
	Attachments []ParsedAttachment
}

// ParsedAttachment is a single decoded attachment or inline resource extracted
// from a parsed message.
type ParsedAttachment struct {
	// Filename is the part's filename, from Content-Disposition or Content-Type.
	Filename string
	// ContentType is the part's MIME media type without parameters.
	ContentType string
	// ContentID is the Content-ID value with any angle brackets removed.
	ContentID string
	// Inline reports whether the part had a Content-Disposition of "inline".
	Inline bool
	// Content is the fully decoded part body.
	Content []byte
}

// Get returns the decoded value of the named top-level header (case-insensitive),
// or the empty string when absent. Encoded-words in the value are decoded to
// UTF-8.
func (pm *ParsedMessage) Get(key string) string {
	v := pm.Header.Get(key)
	if v == "" {
		return ""
	}
	dec, err := DecodeHeaderWord(v)
	if err != nil {
		return v
	}
	return dec
}

// ParseMIME parses raw RFC 5322 / MIME bytes into a ParsedMessage, decoding
// transfer encodings (base64, quoted-printable), walking nested multipart
// containers and separating body text from attachments. It is the inverse, in
// spirit, of Message.Build.
func ParseMIME(raw []byte) (*ParsedMessage, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("nodemailer: parse message: %w", err)
	}
	pm := &ParsedMessage{Header: msg.Header}

	dec := &mime.WordDecoder{}
	if s, err := dec.DecodeHeader(msg.Header.Get("Subject")); err == nil {
		pm.Subject = s
	} else {
		pm.Subject = msg.Header.Get("Subject")
	}
	pm.From = parseHeaderAddrs(msg.Header.Get("From"))
	pm.To = parseHeaderAddrs(msg.Header.Get("To"))
	pm.Cc = parseHeaderAddrs(msg.Header.Get("Cc"))
	if d, err := msg.Header.Date(); err == nil {
		pm.Date = d
	}
	pm.MessageID = strings.Trim(strings.TrimSpace(msg.Header.Get("Message-ID")), "<>")

	if err := pm.walk(textproto.MIMEHeader(msg.Header), msg.Body); err != nil {
		return nil, err
	}
	return pm, nil
}

// walk processes a single MIME entity given its headers and body reader,
// recursing into multipart containers.
func (pm *ParsedMessage) walk(header textproto.MIMEHeader, body io.Reader) error {
	ctype := header.Get("Content-Type")
	if ctype == "" {
		ctype = "text/plain"
	}
	mediaType, params, err := mime.ParseMediaType(ctype)
	if err != nil {
		mediaType = strings.ToLower(strings.TrimSpace(strings.SplitN(ctype, ";", 2)[0]))
		params = map[string]string{}
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return fmt.Errorf("nodemailer: multipart part missing boundary")
		}
		mr := multipart.NewReader(body, boundary)
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("nodemailer: read multipart: %w", err)
			}
			if err := pm.walk(part.Header, part); err != nil {
				_ = part.Close()
				return err
			}
			_ = part.Close()
		}
		return nil
	}

	decoded, err := decodePartBody(header.Get("Content-Transfer-Encoding"), body)
	if err != nil {
		return err
	}

	disposition := header.Get("Content-Disposition")
	dispType := ""
	dispParams := map[string]string{}
	if disposition != "" {
		if dt, dp, err := mime.ParseMediaType(disposition); err == nil {
			dispType = strings.ToLower(dt)
			dispParams = dp
		}
	}
	filename := dispParams["filename"]
	if filename == "" {
		filename = params["name"]
	}
	isAttachment := dispType == "attachment" || dispType == "inline" || filename != ""

	if isAttachment {
		pm.Attachments = append(pm.Attachments, ParsedAttachment{
			Filename:    filename,
			ContentType: mediaType,
			ContentID:   strings.Trim(strings.TrimSpace(header.Get("Content-ID")), "<>"),
			Inline:      dispType == "inline",
			Content:     decoded,
		})
		return nil
	}

	switch mediaType {
	case "text/html":
		pm.HTML += string(decoded)
	default:
		pm.Text += string(decoded)
	}
	return nil
}

// decodePartBody reads and decodes a part body according to its
// Content-Transfer-Encoding.
func decodePartBody(encoding string, body io.Reader) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "base64":
		data, err := io.ReadAll(body)
		if err != nil {
			return nil, err
		}
		clean := strings.Map(func(r rune) rune {
			if r == '\r' || r == '\n' || r == ' ' || r == '\t' {
				return -1
			}
			return r
		}, string(data))
		return base64.StdEncoding.DecodeString(clean)
	case "quoted-printable":
		return io.ReadAll(quotedprintable.NewReader(body))
	default:
		return io.ReadAll(body)
	}
}

// parseHeaderAddrs parses an address-list header value into Address values,
// returning nil on any error.
func parseHeaderAddrs(v string) []Address {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	list, err := mail.ParseAddressList(v)
	if err != nil {
		return nil
	}
	out := make([]Address, 0, len(list))
	for _, a := range list {
		out = append(out, Address{Name: a.Name, Address: a.Address})
	}
	return out
}
