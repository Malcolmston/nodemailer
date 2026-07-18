package nodemailer

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
)

// DNSRecordName returns the DNS host name at which this DKIM key's public record
// must be published, i.e. "<selector>._domainkey.<domain>". It returns an error
// when the selector or domain is unset.
func (d *DKIM) DNSRecordName() (string, error) {
	if d.Selector == "" || d.Domain == "" {
		return "", ErrDKIMConfig
	}
	return d.Selector + "._domainkey." + d.Domain, nil
}

// DNSRecord returns the value of the TXT record to publish for this DKIM key,
// derived from the configured private key's public half. The record has the
// form "v=DKIM1; k=rsa; p=<base64 SubjectPublicKeyInfo>", which verifiers fetch
// to validate a DKIM-Signature. It returns an error when the private key is
// unset or cannot be marshalled.
func (d *DKIM) DNSRecord() (string, error) {
	if d.PrivateKey == nil {
		return "", ErrDKIMConfig
	}
	der, err := x509.MarshalPKIXPublicKey(&d.PrivateKey.PublicKey)
	if err != nil {
		return "", fmt.Errorf("nodemailer: marshal DKIM public key: %w", err)
	}
	p := base64.StdEncoding.EncodeToString(der)
	return "v=DKIM1; k=rsa; p=" + p, nil
}
