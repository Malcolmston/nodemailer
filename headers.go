package nodemailer

import "strings"

// Priority expresses a message's importance, mapping to the X-Priority,
// X-MSMail-Priority and Importance headers.
type Priority int

const (
	// PriorityNormal leaves priority headers unset (the default).
	PriorityNormal Priority = iota
	// PriorityHigh marks the message as high priority.
	PriorityHigh
	// PriorityLow marks the message as low priority.
	PriorityLow
)

// headers returns the priority header fields for p, or nil for PriorityNormal.
func (p Priority) headers() []Header {
	switch p {
	case PriorityHigh:
		return []Header{
			{"X-Priority", "1 (Highest)"},
			{"X-MSMail-Priority", "High"},
			{"Importance", "High"},
		}
	case PriorityLow:
		return []Header{
			{"X-Priority", "5 (Lowest)"},
			{"X-MSMail-Priority", "Low"},
			{"Importance", "Low"},
		}
	default:
		return nil
	}
}

// SetPriority sets the message priority (see Priority).
func (m *Message) SetPriority(p Priority) *Message { m.Priority = p; return m }

// WithDKIM configures DKIM signing for the message and returns it for chaining.
func (m *Message) WithDKIM(d *DKIM) *Message { m.DKIM = d; return m }

// SetInReplyTo sets the In-Reply-To header from a Message-ID (without angle
// brackets).
func (m *Message) SetInReplyTo(messageID string) *Message {
	m.InReplyTo = strings.Trim(messageID, "<>")
	return m
}

// AddReferences appends Message-IDs (without angle brackets) to the References
// header, used for threading.
func (m *Message) AddReferences(messageIDs ...string) *Message {
	for _, id := range messageIDs {
		m.References = append(m.References, strings.Trim(id, "<>"))
	}
	return m
}

// AddListHeader appends an RFC 2369 List-* header. The key may be given with or
// without the "List-" prefix (e.g. "Unsubscribe" or "List-Unsubscribe").
func (m *Message) AddListHeader(key, value string) *Message {
	if !strings.HasPrefix(strings.ToLower(key), "list-") {
		key = "List-" + key
	}
	m.ListHeaders = append(m.ListHeaders, Header{Key: key, Value: value})
	return m
}

// SetListUnsubscribe sets the List-Unsubscribe header from one or more URIs
// (mailto: or https:). Each URI is wrapped in angle brackets per RFC 2369.
func (m *Message) SetListUnsubscribe(uris ...string) *Message {
	parts := make([]string, len(uris))
	for i, u := range uris {
		parts[i] = "<" + strings.Trim(u, "<>") + ">"
	}
	return m.AddListHeader("Unsubscribe", strings.Join(parts, ", "))
}

// SetListUnsubscribePost enables RFC 8058 one-click unsubscribe by adding the
// List-Unsubscribe-Post header. Call SetListUnsubscribe as well to supply the
// unsubscribe URIs.
func (m *Message) SetListUnsubscribePost() *Message {
	return m.AddListHeader("Unsubscribe-Post", "List-Unsubscribe=One-Click")
}

// referencesValue renders the References header value from the stored IDs.
func (m *Message) referencesValue() string {
	parts := make([]string, len(m.References))
	for i, id := range m.References {
		parts[i] = "<" + id + ">"
	}
	return strings.Join(parts, " ")
}
