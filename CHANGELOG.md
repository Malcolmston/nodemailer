# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
