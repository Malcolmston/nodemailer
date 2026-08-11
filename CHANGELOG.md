# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0] - 2026-08-11

### Security

- **Email header injection (GHSA-pgqx-3gm6-xx62, CWE-93, medium).** A bare CR or
  LF in a header-valued field was written into the message verbatim, so a value
  such as `Subject: "Innocent\r\nBcc: attacker@example.com"` ended the `Subject`
  field early and appended a real `Bcc` header — and, after a second CRLF, a
  forged message body. Any application passing user-controlled text (a ticket
  title, a display name, an order reference) to `SetSubject` or `AddHeader`
  could have arbitrary headers written on its behalf, which is enough to
  redirect replies or exfiltrate the body to an added recipient.

  `Message.Build` now validates every field that reaches a header value —
  `Subject`, custom `Headers` (name and value), `List-*` headers, display names
  and addr-specs of `From`/`To`/`Cc`/`Bcc`/`Reply-To` and of address groups,
  `Message-ID`, `In-Reply-To`, `References`, the MIME boundary, attachment
  filenames/content types/content IDs, alternative content types and the
  calendar method — and fails with the new sentinel `ErrHeaderInjection` when
  one carries a CR or LF. `Address.Validate` likewise rejects a newline in a
  display name.

### Added

- `ErrHeaderInjection`, the sentinel returned by `Build` for a newline-bearing
  header value.
- `SanitizeHeaderValue`, which folds each run of CR/LF into a single space. It
  is the migration path for callers that must accept arbitrary untrusted text:
  sanitise first, then set the header. Non-ASCII header text continues to be
  carried as an RFC 2047 encoded-word (see `EncodeWord`), which is the only
  correct way to do so — raw bytes never were.

### Changed (breaking)

- `Message.Build` now returns an error instead of a message when a header value
  contains a CR or LF. Code that previously "worked" by relying on that
  behaviour — deliberately or not — now fails loudly. This is intentional: RFC
  5322 forbids bare CR/LF in a field body, so no legitimate header value is
  affected. Wrap the value in `SanitizeHeaderValue` to keep building.

## [0.3.0] - 2026-07-18

Further Nodemailer parity. Still standard-library only: no third-party imports,
no cgo. All output remains deterministic where the underlying feature allows.

### Added

- **Well-known SMTP services** (`Service`, `WellKnownService`,
  `WellKnownServiceNames`, `NewServiceSMTP`): configure an `SMTPTransport` by
  provider name (Gmail, Outlook365, Yahoo, SendGrid, SES, ...), mirroring
  nodemailer's `service` option and well-known/services table.
- **HTML-to-text generation** (`HTMLToText`, `Message.GenerateTextFromHTML`):
  derive a `text/plain` alternative from an HTML body, mirroring
  nodemailer's `generateTextFromHtml`.
- **AMP and Apple Watch bodies** (`Message.SetAMP`, `Message.SetWatchHTML`):
  convenience setters for the `amp` and `watchHtml` alternatives.
- **Stream transport** (`StreamTransport`, `NewStreamTransport`): write encoded
  messages to any `io.Writer`, mirroring nodemailer's `streamTransport`.
- **Address utilities** (`Address.Local`, `Address.Domain`, `Address.Equal`,
  `NormalizeAddress`, `FormatAddressList`): mailbox part accessors and
  case-correct comparison/normalization.
- **MIME word helpers** (`EncodeWord`, `DecodeHeaderWord`) and a public
  Message-ID generator (`GenerateMessageID`), mirroring libmime's
  `mimeWordEncode` / `mimeWordsDecode`.
- **DKIM DNS record helpers** (`DKIM.DNSRecordName`, `DKIM.DNSRecord`): emit the
  `<selector>._domainkey.<domain>` host and the `v=DKIM1; k=rsa; p=...` TXT
  value to publish for a signing key.
- **MIME parsing** (`ParsedMessage`, `ParsedAttachment`, `ParseMIME`,
  `ParsedMessage.Get`): decode raw RFC 5322/MIME bytes back into headers,
  addresses, bodies and attachments, a lightweight counterpart to mailparser.

## [0.2.0] - 2026-07-17

Expanded the library toward broader Nodemailer parity. Still standard-library
only: no third-party imports, no cgo.

### Added

- **DKIM signing** (`DKIM`, `ParseRSAPrivateKey`): RSA-SHA256 with relaxed and
  simple header/body canonicalization, emitting a `DKIM-Signature` header.
- **OAuth2 / XOAUTH2** SMTP authentication (`XOAuth2Auth`, `XOAuth2Token`).
- **SMTP connection pool** (`Pool`): bounded, reusable connections with a
  configurable maximum number of messages per connection.
- **Sendmail transport** (`SendmailTransport`): pipes messages to a local
  sendmail-compatible binary via `os/exec`.
- **Connection verification** (`SMTPTransport.Verify`, `Pool.Verify`): dial +
  EHLO + optional auth with no message sent.
- **Attachments and embeds from more sources**: `AttachFile`, `AttachReader`,
  `AttachURL`, `EmbedFile`, `EmbedReader`, with content-type sniffing.
- **Named address groups** (`AddressGroup`, `ParseAddressGroup`,
  `Message.AddToGroup`, `Message.AddCcGroup`).
- **List-\* headers** (`SetListUnsubscribe`, `SetListUnsubscribePost`,
  `AddListHeader`), including RFC 8058 one-click unsubscribe.
- **Priority headers** (`Priority`, `SetPriority`): `X-Priority`,
  `X-MSMail-Priority`, `Importance`.
- **Threading headers** (`SetInReplyTo`, `AddReferences`).
- **Extra body alternatives** (`AddAlternative`) and **iCal events**
  (`ICalEvent`) rendered inside `multipart/alternative`.
- **DSN options** (`DSNOptions`): RFC 3461 `RET`/`ENVID`/`NOTIFY`/`ORCPT`
  parameters on SMTP delivery.

## [0.1.0]

### Added

- Initial release: fluent `Message` builder, MIME encoder (multipart
  alternative/related/mixed, quoted-printable, base64, RFC 2047), address
  parsing/validation, and SMTP, in-memory and JSON transports.
