// Library content for the nodemailer documentation site. Mirrors the shape used
// by the malcolmston/go landing site's data.ts so the sibling sites stay in sync.
export interface Lib {
  id: string; name: string; icon: string; accent: string; pkg: string; node: string;
  repo: string; docs: string; tagline: string; blurb: string; tags: string[];
  features: string[]; node_code: string; go_code: string; integrate: string;
}

export const NODE_ACCENT = '#8cc84b';

export const NODEMAILER: Lib = {
  id:"nodemailer", name:"Nodemailer", icon:'<i class="fa-solid fa-envelope"></i>', accent:"#34b3a0",
  pkg:"github.com/malcolmston/nodemailer", node:"nodemailer/nodemailer",
  repo:"https://github.com/malcolmston/nodemailer", docs:"https://malcolmston.github.io/nodemailer/",
  tagline:"Nodemailer-style email composition and SMTP sending for Go.",
  blurb:"A from-scratch, standard-library-only Go take on Node.js's Nodemailer: compose rich MIME email with a "+
    "fluent builder and deliver it through pluggable transports, with no cgo and no third-party dependencies. "+
    "A Message assembles From/To/Cc/Bcc/Reply-To, a subject, plain-text and HTML bodies, custom headers, file "+
    "attachments and inline (CID) images; addresses are parsed and validated through net/mail. Build emits "+
    "correct MIME — multipart/alternative for text+html, wrapped in multipart/related for inline resources and "+
    "multipart/mixed for attachments, with quoted-printable bodies, base64 attachments, RFC 2047 encoded words, "+
    "CRLF endings and header folding. A Transport interface carries the bytes: SMTPTransport speaks PLAIN auth "+
    "over STARTTLS or implicit TLS, while MemoryTransport and JSONTransport capture messages for tests. Set the "+
    "Date, Message-ID and boundary explicitly for byte-for-byte deterministic output.",
  tags:["Message builder","net/mail addresses","MIME encoder","multipart/alternative","multipart/related","multipart/mixed","quoted-printable","base64","RFC 2047","SMTP + STARTTLS","MemoryTransport","deterministic"],
  features:[
    "Fluent <code>Message</code> builder — <code>New()</code> plus <code>SetFrom</code>, <code>AddTo</code>, <code>AddCc</code>, <code>AddBcc</code>, <code>AddReplyTo</code>, <code>SetSubject</code>, <code>SetText</code>, <code>SetHTML</code>",
    "Attachments &amp; inline images — <code>Attach</code>, <code>AttachBytes</code> and <code>Embed</code> (cid: resources) over the <code>Attachment</code> struct",
    "Address parsing &amp; validation via <code>ParseAddress</code> / <code>ParseAddressList</code> on <code>net/mail</code>, with deferred errors surfaced by <code>Err</code>",
    "Automatic MIME structure from <code>Build</code> — single part, <code>multipart/alternative</code>, wrapped in <code>multipart/related</code> for inline and <code>multipart/mixed</code> for attachments",
    "Correct encoding — quoted-printable bodies, base64 attachments, RFC 2047 encoded words for non-ASCII subjects/names/filenames, CRLF endings and header folding",
    "Pluggable <code>Transport</code> interface — <code>Send(from, to, raw)</code> decouples composition from delivery",
    "<code>SMTPTransport</code> over <code>net/smtp</code> — PLAIN auth, implicit <code>TLS</code> or <code>STARTTLS</code>, custom <code>TLSConfig</code> and <code>LocalName</code>",
    "Test transports — <code>MemoryTransport</code> captures raw messages (<code>Last</code>) and <code>JSONTransport</code> records a JSON serialisation of each send",
    "<code>Transporter.SendMail</code> mirrors nodemailer's flow, deriving the <code>Envelope</code> and returning <code>Info</code> with <code>MessageID</code>, <code>Envelope</code> and <code>Raw</code>",
    "Deterministic output — <code>SetDate</code>, <code>SetMessageID</code> and <code>SetBoundary</code> make <code>Build</code> byte-for-byte stable for golden tests",
    "Zero dependencies — pure Go standard library, no cgo, nothing to audit but the toolchain"
  ],
  node_code:
`import nodemailer from "nodemailer";

const transporter = nodemailer.createTransport({
  host: "smtp.example.com",
  port: 587,
  secure: false,
  auth: { user: "ada", pass: "secret" },
});

const info = await transporter.sendMail({
  from: "Ada Lovelace <ada@example.com>",
  to: "grace@example.com",
  subject: "Progress report",
  text: "Plain-text version.",
  html: "<p>HTML version</p>",
});
console.log(info.messageId);`,
  go_code:
`import "github.com/malcolmston/nodemailer"

msg := nodemailer.New().
	SetFrom("Ada Lovelace <ada@example.com>").
	AddTo("grace@example.com").
	SetSubject("Progress report").
	SetText("Plain-text version.").
	SetHTML("<p>HTML version</p>")

tr := nodemailer.NewTransporter(&nodemailer.MemoryTransport{})
info, _ := tr.SendMail(msg)
fmt.Println(info.MessageID)`,
  integrate:
`<span class="tok-c">// Compose a rich message: text + HTML alternative, an inline logo</span>
<span class="tok-c">// referenced from the HTML via cid:logo, and a PDF attachment.</span>
msg := nodemailer.New().
	SetFrom("Ada Lovelace <ada@example.com>").
	AddTo("Grace Hopper <grace@example.com>", "team@example.com").
	AddCc("carl@example.com").
	SetSubject("Progress report").
	SetText("Plain-text version of the message.").
	SetHTML(\`<p>See the logo: <img src="cid:logo"></p>\`).
	Embed("logo", "logo.png", "image/png", pngBytes).
	AttachBytes("report.pdf", "application/pdf", pdfBytes)

<span class="tok-c">// Deliver over authenticated SMTP with STARTTLS; SendMail builds the</span>
<span class="tok-c">// MIME bytes, derives the envelope and returns Info.</span>
tr := nodemailer.NewTransporter(&nodemailer.SMTPTransport{
	Host: "smtp.example.com", Port: 587,
	Username: "ada", Password: "secret", STARTTLS: true,
})
info, err := tr.SendMail(msg)
if err != nil {
	log.Fatal(err)
}
log.Printf("sent %s to %v", info.MessageID, info.Envelope.To)

<span class="tok-c">// For golden tests, pin Date/Message-ID/Boundary and capture the raw</span>
<span class="tok-c">// bytes with an in-memory transport instead of a live server.</span>
mem := &nodemailer.MemoryTransport{}
_, _ = nodemailer.NewTransporter(mem).SendMail(
	msg.SetDate(time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)).
		SetMessageID("report-42@example.com").
		SetBoundary("BOUNDARY"))
captured, _ := mem.Last() <span class="tok-c">// captured.Raw holds the full RFC 5322 message</span>`
};
