package nodemailer

// Envelope is the SMTP envelope: the sender and the full recipient list
// actually used for delivery (To + Cc + Bcc).
type Envelope struct {
	From string
	To   []string
}

// Info describes the result of a successful send.
type Info struct {
	// MessageID is the Message-ID header value (without angle brackets).
	MessageID string
	// Envelope is the SMTP envelope used for delivery.
	Envelope Envelope
	// Raw is the encoded message that was handed to the transport.
	Raw []byte
}

// Transporter builds messages and delivers them through a Transport, mirroring
// nodemailer's createTransport().sendMail() flow.
type Transporter struct {
	Transport Transport
}

// NewTransporter returns a Transporter backed by t.
func NewTransporter(t Transport) *Transporter {
	return &Transporter{Transport: t}
}

// SendMail encodes m and delivers it through the configured transport. It
// returns Info describing the delivery, or an error if building or sending
// fails.
func (tr *Transporter) SendMail(m *Message) (Info, error) {
	raw, err := m.Build()
	if err != nil {
		return Info{}, err
	}

	env := Envelope{
		From: m.From.Address,
		To:   m.Recipients(),
	}

	if err := tr.Transport.Send(env.From, env.To, raw); err != nil {
		return Info{}, err
	}

	return Info{
		MessageID: extractMessageID(raw),
		Envelope:  env,
		Raw:       raw,
	}, nil
}

// extractMessageID reads the Message-ID value (without angle brackets) from the
// encoded message headers.
func extractMessageID(raw []byte) string {
	const key = "Message-ID: <"
	s := string(raw)
	idx := indexHeader(s, "Message-ID: <")
	if idx < 0 {
		return ""
	}
	rest := s[idx+len(key):]
	if end := indexByte(rest, '>'); end >= 0 {
		return rest[:end]
	}
	return ""
}

// indexHeader finds a header prefix at the start of a line.
func indexHeader(s, prefix string) int {
	// Header block ends at the first blank line.
	for i := 0; i+len(prefix) <= len(s); i++ {
		if (i == 0 || s[i-1] == '\n') && s[i:i+len(prefix)] == prefix {
			return i
		}
		if i+1 < len(s) && s[i] == '\r' && s[i+1] == '\n' && i+3 < len(s) && s[i+2] == '\r' && s[i+3] == '\n' {
			break
		}
	}
	return -1
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
