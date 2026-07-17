package nodemailer

import "strings"

// ICalEvent describes a calendar invitation carried as a text/calendar
// alternative body, mirroring nodemailer's icalEvent option.
type ICalEvent struct {
	// Method is the iCalendar method (REQUEST, PUBLISH, CANCEL, REPLY). It
	// defaults to PUBLISH and is emitted as the Content-Type method parameter.
	Method string
	// Content is the raw iCalendar document (a VCALENDAR block).
	Content string
	// Filename is the name used when the event is presented as an attachment;
	// it defaults to "invite.ics".
	Filename string
}

// method returns the effective iCalendar method, defaulting to PUBLISH.
func (e *ICalEvent) method() string {
	if e.Method == "" {
		return "PUBLISH"
	}
	return strings.ToUpper(e.Method)
}

// part builds the text/calendar MIME leaf for the event, base64-encoded so that
// arbitrary CRLF-delimited calendar data survives transport intact.
func (e *ICalEvent) part() *mimePart {
	ct := "text/calendar; charset=utf-8; method=" + e.method()
	return &mimePart{
		headers: []Header{
			{"Content-Type", ct},
			{"Content-Transfer-Encoding", "base64"},
		},
		body: base64Wrap([]byte(e.Content)),
	}
}
