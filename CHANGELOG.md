# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
