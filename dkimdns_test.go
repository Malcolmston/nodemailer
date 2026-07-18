package nodemailer

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"strings"
	"testing"
)

func TestDKIMDNSRecordName(t *testing.T) {
	d := &DKIM{Domain: "example.com", Selector: "mail"}
	name, err := d.DNSRecordName()
	if err != nil {
		t.Fatal(err)
	}
	if name != "mail._domainkey.example.com" {
		t.Errorf("name = %q", name)
	}

	if _, err := (&DKIM{Domain: "example.com"}).DNSRecordName(); err == nil {
		t.Error("expected error when selector missing")
	}
}

func TestDKIMDNSRecord(t *testing.T) {
	key := mustKey(t)
	d := &DKIM{Domain: "example.com", Selector: "mail", PrivateKey: key}
	rec, err := d.DNSRecord()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rec, "v=DKIM1; k=rsa; p=") {
		t.Fatalf("record prefix wrong: %q", rec)
	}
	p := strings.TrimPrefix(rec, "v=DKIM1; k=rsa; p=")
	der, err := base64.StdEncoding.DecodeString(p)
	if err != nil {
		t.Fatalf("p= is not valid base64: %v", err)
	}
	pub, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		t.Fatalf("p= is not a valid public key: %v", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		t.Fatalf("public key is not RSA: %T", pub)
	}
	if rsaPub.N.Cmp(key.PublicKey.N) != 0 {
		t.Error("published public key does not match the signing key")
	}

	// Missing key is an error.
	if _, err := (&DKIM{Domain: "d", Selector: "s"}).DNSRecord(); err == nil {
		t.Error("expected error when private key missing")
	}
}
