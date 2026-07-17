// Package nodemailer is a Nodemailer-style email composition and SMTP sending
// library implemented entirely with the Go standard library.
//
// It provides three layers that compose cleanly:
//
//   - A Message builder for assembling an email from its parts: From, To, Cc,
//     Bcc, Reply-To, Subject, plain-text and HTML bodies, custom headers, an
//     explicit Date and Message-ID, file attachments and inline (CID) images.
//   - A MIME encoder that turns a Message into RFC 5322 / MIME bytes.
//   - A Transport interface with concrete SMTP, in-memory and JSON transports,
//     driven by a Transporter that mirrors nodemailer's sendMail flow.
//
// # Building a message
//
// Messages are constructed with New and the fluent setters, or by populating
// the exported struct fields directly. Address setters parse and validate
// their inputs using net/mail; the first error is deferred and surfaced by
// Build (or Err):
//
//	msg := nodemailer.New().
//		SetFrom("Ada Lovelace <ada@example.com>").
//		AddTo("Grace Hopper <grace@example.com>", "team@example.com").
//		SetSubject("Progress report").
//		SetText("Plain-text body").
//		SetHTML("<p>HTML body</p>")
//
// Addresses accept the usual forms: a bare addr-spec ("a@b.com"), a named
// address ("Name <a@b.com>") and comma-separated lists. Validation rejects
// clearly-invalid values such as missing "@", empty local or domain parts and
// domains without a dot.
//
// # MIME structure
//
// Build chooses the correct MIME structure automatically:
//
//   - text only or HTML only: a single text/plain or text/html part.
//   - text and HTML: multipart/alternative.
//   - inline (CID) resources: the body is wrapped in multipart/related.
//   - regular attachments: the whole thing is wrapped in multipart/mixed.
//
// Text bodies are quoted-printable encoded; attachments are base64 encoded and
// wrapped at 76 columns. Non-ASCII subjects, display names and filenames use
// RFC 2047 encoded words. All line endings are CRLF and long header lines are
// folded at whitespace.
//
// Output is deterministic when Date, MessageID and Boundary are all set
// explicitly, which makes the encoder straightforward to unit-test:
//
//	msg.SetDate(time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)).
//		SetMessageID("fixed@example.com").
//		SetBoundary("BOUNDARY")
//	raw, err := msg.Build()
//
// # Transports
//
// A Transport delivers encoded bytes to recipients:
//
//	type Transport interface {
//		Send(from string, to []string, raw []byte) error
//	}
//
// SMTPTransport speaks SMTP via net/smtp with optional PLAIN authentication and
// either implicit TLS or STARTTLS. MemoryTransport captures messages in memory
// for tests, and JSONTransport records a JSON serialisation of each message.
//
// A Transporter ties a Message to a Transport:
//
//	tr := nodemailer.NewTransporter(&nodemailer.MemoryTransport{})
//	info, err := tr.SendMail(msg)
//
// SendMail builds the MIME bytes, derives the SMTP envelope (sender plus the
// combined To/Cc/Bcc recipient list) and returns Info containing the
// Message-ID, envelope and raw bytes.
package nodemailer
