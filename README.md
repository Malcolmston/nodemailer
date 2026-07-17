# nodemailer

A Nodemailer-style email composition and SMTP sending library for Go, built
**entirely on the standard library** — no third-party modules, no cgo.

- Fluent `Message` builder: From, To/Cc/Bcc, Reply-To, Subject, text + HTML
  bodies, custom headers, Date, Message-ID, attachments and inline (CID) images.
- Address parsing and validation via `net/mail` ("Name <a@b.com>", comma lists),
  plus RFC 5322 named address **groups** (`Team: a@x, b@y;`).
- Correct MIME encoding: `multipart/alternative` for text+html (and any extra
  body alternatives or a `text/calendar` iCal event), wrapped in
  `multipart/related` for inline images and `multipart/mixed` for attachments;
  quoted-printable bodies, base64 attachments, RFC 2047 encoded words for
  non-ASCII subjects/names/filenames, CRLF line endings and header folding.
- Attachments and embeds from in-memory bytes, a **file path**, an **`io.Reader`**
  or a **URL** (fetched over HTTP), with content-type sniffing.
- Message metadata: `List-*` headers (incl. RFC 8058 one-click
  `List-Unsubscribe`), `X-Priority`/`Importance`, and `In-Reply-To`/`References`
  threading.
- **DKIM** signing (RSA-SHA256, relaxed/simple canonicalization).
- Pluggable `Transport` interface: SMTP (PLAIN **or OAuth2/XOAUTH2** auth,
  STARTTLS/TLS, DSN options, `Verify()`), a bounded reusable **connection
  `Pool`**, a **sendmail** transport, and in-memory/JSON test transports.
- Deterministic output when Date, Message-ID and boundary are fixed.

## Install

```sh
go get github.com/malcolmston/nodemailer
```

Requires Go 1.24+.

## Quick start

```go
package main

import (
	"log"

	"github.com/malcolmston/nodemailer"
)

func main() {
	msg := nodemailer.New().
		SetFrom("Ada Lovelace <ada@example.com>").
		AddTo("Grace Hopper <grace@example.com>", "team@example.com").
		AddCc("carl@example.com").
		SetSubject("Progress report").
		SetText("Plain-text version of the message.").
		SetHTML(`<p>HTML version with an image: <img src="cid:logo"></p>`).
		Embed("logo", "logo.png", "image/png", pngBytes).
		AttachBytes("report.pdf", "application/pdf", pdfBytes)

	transport := &nodemailer.SMTPTransport{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "ada",
		Password: "secret",
		STARTTLS: true,
	}

	info, err := nodemailer.NewTransporter(transport).SendMail(msg)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("sent %s to %v", info.MessageID, info.Envelope.To)
}
```

## Testing without a server

`MemoryTransport` captures the raw message so you can assert on it, and
`JSONTransport` records a JSON serialisation of each send:

```go
mem := &nodemailer.MemoryTransport{}
info, _ := nodemailer.NewTransporter(mem).SendMail(msg)

captured, _ := mem.Last()
// captured.Raw holds the full RFC 5322 bytes.
_ = info
```

## Deterministic MIME output

Set the Date, Message-ID and boundary explicitly to get byte-for-byte stable
output (useful for golden tests):

```go
raw, err := nodemailer.New().
	SetFrom("ada@example.com").
	AddTo("grace@example.com").
	SetSubject("Hi").
	SetText("Hello").
	SetHTML("<p>Hi</p>").
	SetDate(time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)).
	SetMessageID("id@example.com").
	SetBoundary("BOUNDARY").
	Build()
```

## OAuth2 (XOAUTH2) authentication

```go
transport := &nodemailer.SMTPTransport{
	Host:     "smtp.gmail.com",
	Port:     587,
	STARTTLS: true,
	Auth:     nodemailer.XOAuth2Auth("ada@gmail.com", accessToken),
}
```

## Connection pooling and verification

```go
pool := &nodemailer.Pool{
	Transport:      transport,
	MaxConnections: 5,
	MaxMessages:    100,
}
defer pool.Close()

if err := pool.Verify(); err != nil { // dial + EHLO + auth, no send
	log.Fatal(err)
}
info, err := nodemailer.NewTransporter(pool).SendMail(msg)
```

## DKIM signing

```go
key, _ := nodemailer.ParseRSAPrivateKey(pemBytes)
msg.WithDKIM(&nodemailer.DKIM{
	Domain:     "example.com",
	Selector:   "mail",
	PrivateKey: key, // RSA-SHA256, relaxed/relaxed by default
})
raw, _ := msg.Build() // raw now begins with a DKIM-Signature header
```

## Sendmail transport

```go
info, err := nodemailer.NewTransporter(&nodemailer.SendmailTransport{
	Path: "/usr/sbin/sendmail",
}).SendMail(msg)
```

## Attachments from a file, reader or URL

```go
msg.AttachFile("/path/to/report.pdf").
	AttachReader("data.bin", "", reader).
	AttachURL("https://example.com/chart.png").
	EmbedFile("logo", "/path/to/logo.png") // inline cid:logo
```

## License

See repository.
