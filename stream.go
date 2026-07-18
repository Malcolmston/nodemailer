package nodemailer

import (
	"io"
	"sync"
)

// StreamTransport writes each fully-encoded message to an io.Writer instead of
// delivering it over the network, mirroring nodemailer's streamTransport. It is
// useful for piping messages to a file, a buffer or another process. A blank
// line is written between successive messages so a single stream remains
// readable.
//
// StreamTransport is safe for concurrent use.
type StreamTransport struct {
	// Writer receives the raw bytes of every sent message. It must be non-nil.
	Writer io.Writer

	mu    sync.Mutex
	count int
}

// NewStreamTransport returns a StreamTransport that writes encoded messages to
// w.
func NewStreamTransport(w io.Writer) *StreamTransport {
	return &StreamTransport{Writer: w}
}

// Send writes the raw message bytes to the configured Writer. The from and to
// arguments are ignored; the envelope is already encoded in the message
// headers. It returns any error reported by the underlying Writer.
func (t *StreamTransport) Send(from string, to []string, raw []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.count > 0 {
		if _, err := io.WriteString(t.Writer, crlf); err != nil {
			return err
		}
	}
	t.count++
	_, err := t.Writer.Write(raw)
	return err
}
