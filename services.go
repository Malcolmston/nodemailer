package nodemailer

import (
	"fmt"
	"sort"
	"strings"
)

// Service describes the SMTP connection settings for a well-known email
// provider, mirroring an entry in nodemailer's well-known/services.json. It lets
// callers configure an SMTPTransport by provider name instead of remembering
// host, port and TLS mode.
type Service struct {
	// Name is the canonical provider name, e.g. "Gmail".
	Name string
	// Host is the SMTP submission host, e.g. "smtp.gmail.com".
	Host string
	// Port is the SMTP submission port, e.g. 465 or 587.
	Port int
	// Secure reports whether the connection uses implicit TLS from the start
	// (typically port 465). When false, callers should use STARTTLS on the
	// submission port (typically 587).
	Secure bool
}

// wellKnownServices maps a lower-cased lookup key (canonical name or alias) to a
// provider's SMTP settings. Values follow nodemailer's well-known service table.
var wellKnownServices = map[string]Service{
	"gmail":      {Name: "Gmail", Host: "smtp.gmail.com", Port: 465, Secure: true},
	"googlemail": {Name: "Gmail", Host: "smtp.gmail.com", Port: 465, Secure: true},
	"outlook365": {Name: "Outlook365", Host: "smtp.office365.com", Port: 587, Secure: false},
	"hotmail":    {Name: "Hotmail", Host: "smtp-mail.outlook.com", Port: 587, Secure: false},
	"outlook":    {Name: "Outlook365", Host: "smtp.office365.com", Port: 587, Secure: false},
	"yahoo":      {Name: "Yahoo", Host: "smtp.mail.yahoo.com", Port: 465, Secure: true},
	"icloud":     {Name: "iCloud", Host: "smtp.mail.me.com", Port: 587, Secure: false},
	"zoho":       {Name: "Zoho", Host: "smtp.zoho.com", Port: 465, Secure: true},
	"sendgrid":   {Name: "SendGrid", Host: "smtp.sendgrid.net", Port: 587, Secure: false},
	"mailgun":    {Name: "Mailgun", Host: "smtp.mailgun.org", Port: 587, Secure: false},
	"postmark":   {Name: "Postmark", Host: "smtp.postmarkapp.com", Port: 587, Secure: false},
	"ses":        {Name: "SES", Host: "email-smtp.us-east-1.amazonaws.com", Port: 465, Secure: true},
	"fastmail":   {Name: "FastMail", Host: "smtp.fastmail.com", Port: 465, Secure: true},
	"yandex":     {Name: "Yandex", Host: "smtp.yandex.ru", Port: 465, Secure: true},
	"mailtrap":   {Name: "Mailtrap", Host: "live.smtp.mailtrap.io", Port: 587, Secure: false},
	"protonmail": {Name: "ProtonMail", Host: "smtp.protonmail.ch", Port: 587, Secure: false},
	"gandi":      {Name: "Gandi", Host: "mail.gandi.net", Port: 587, Secure: false},
}

// WellKnownService looks up the SMTP settings for a provider by name (or alias),
// case-insensitively. It reports whether the provider is known, mirroring
// nodemailer's getWellKnownService.
func WellKnownService(name string) (Service, bool) {
	svc, ok := wellKnownServices[strings.ToLower(strings.TrimSpace(name))]
	return svc, ok
}

// WellKnownServiceNames returns the canonical names of all supported well-known
// services in alphabetical order, with duplicates (from aliases) removed.
func WellKnownServiceNames() []string {
	seen := map[string]bool{}
	var names []string
	for _, svc := range wellKnownServices {
		if !seen[svc.Name] {
			seen[svc.Name] = true
			names = append(names, svc.Name)
		}
	}
	sort.Strings(names)
	return names
}

// NewServiceSMTP builds an SMTPTransport pre-configured for a well-known
// provider identified by service name (e.g. "Gmail"), using the given username
// and password for PLAIN authentication. It returns an error if the service is
// unknown. The returned transport can be adjusted further before use.
func NewServiceSMTP(service, username, password string) (*SMTPTransport, error) {
	svc, ok := WellKnownService(service)
	if !ok {
		return nil, fmt.Errorf("nodemailer: unknown mail service %q", service)
	}
	t := &SMTPTransport{
		Host:     svc.Host,
		Port:     svc.Port,
		Username: username,
		Password: password,
	}
	if svc.Secure {
		t.TLS = true
	} else {
		t.STARTTLS = true
	}
	return t, nil
}
