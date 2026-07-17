package nodemailer

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

// testDKIMKey is a throwaway 2048-bit RSA key used only for deterministic tests.
const testDKIMKey = `-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQDTSJdXr+eaRzYf
j33cRWooGMFfsAxdf8KUTnQd3jezuYvvcHP5aHgKUZGIvvUCBLLrV28dTr0jMVoa
6BzWjFtu12MMwG/uw0T9CP0pZ51mipX+T8kioXbhzAJhrH3cGHHw6uIzpln6rIHW
2YiPwXaoZD312BxtpoWfe29+lQ9m+SlzNUqqLEgkX3xkJwBib8le530+CRWHoWRj
xpNwbanVBQOXuIWIcDQrTyl/OOWIXcxhPwsWOGnOZi8ne6j8GYMuAe0JDMbEc1no
tOZJ4xHfi3pjBFt+qRCaNla10/sFP4r+1tLDe3qulVvvbShgeDtmRvy7Eus6SKBo
ZC2d/L/JAgMBAAECggEAWU0xSoq65WZ75dMMa3GbcN8lvH/4efxqDa0rPwSRnpEq
KEXrfts9AX+Ad7/eZr/9r6MA/+4r2xgc8Ypxxe0FKFA5OUaNCOnX2utVtY5p5NFm
nFq0tMJyMPW9e/MgL0rVvfJJMXN6TI1lQ54mLjyjHoFf/u0c9uuPRt0xPttZ6zdY
RiaP9dg8xTR3SnzQrrHBohHd/T3fn9dnK8Rsqn/ji7BcNz0NdU4L1GTI92Dp+Ifk
mFbHzCRZpeiJu0/fNNkgQ0BCKS6icgSqykxmELQIwxOsg6tsCuSlHKVAhvVcEc+3
faRD85breCNavk6rmBSk40yztMge3kitFtjjntrYEQKBgQDqqlb674AvMQn53wD3
C421BUn478KgaphiBHbylS/E+KhC4d1GAvyNB17ZMa97bN2qdmoXjEgKcYBtNT4a
Zs/V8JwbX1T5nuVa2MJSevZowyjSg0yxiSuXS+8TRFbDOcsx6GeY576/jpBTVb+h
6jUiOlFABwSw0GqdbK7D7WbpDwKBgQDmfg6STc0V1Pjm0pgzJ3feaoOaH/TJykYg
LQCDjRZbWHaOlg17BGccRJNKdeNOlxpGzwm9WrrmpKGXxIim08KDY54IlVduNUYp
Nv4lFJa8C+TSd27EQieGO4OVZfmMdT+L3lHu2kTaoU5QQ9OCoP6NNpeEIRyAORMH
YwO5ZOPZpwKBgQCpju9OXeOnNa3ZqHLQDr8Tv4CVqNhehOcaW9N+sKFVl74spXr2
7Y2CcYLtOONtMVpxoyJBZZFgSmbbgg8fkI44LaT+ekGyJEfg/qJaapLFW86RXWH7
HfwrVCipKUXvxkC2DRFeAIVpcB+Et37CBbLiynSO6QNQpyeCHFejJlSnrQKBgQC2
+Xcj2bNnE2yMAL5mTXyxCily3s96qaLFxDPWOth2p2Fmi+QjtjkMjbvHrpJGP1nS
wGTg9vfMRQEq9A/vL8gIebpo4fVIPe52pXtXgGKw4VhDZCCAmGu7+d7ZaNyUDjfm
FxU/4fIrBUagHVf5KUkqXR4m/AoeGDDs+kNol5jxnQKBgQCqk7Fs4WxGBT8ecfnl
Dj8jcMvh9xfoXQgGSMQSL7xD9XE9mFwqjrXbK6cqY6s/Cv0s5W6YsGkVQdcE2Tnr
KbRVUW7NcUu1MCbSnn5eWKIrtE0WAG+xygMCJDFcuLTfq++SUs+9wHnBmkDZiKW0
bSQMMMeumZkLkJATj4aMXgpoVQ==
-----END PRIVATE KEY-----`

func mustKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := ParseRSAPrivateKey([]byte(testDKIMKey))
	if err != nil {
		t.Fatalf("ParseRSAPrivateKey: %v", err)
	}
	return key
}

func TestParseRSAPrivateKey(t *testing.T) {
	if _, err := ParseRSAPrivateKey([]byte("not a key")); err == nil {
		t.Error("expected error for non-PEM input")
	}
	key := mustKey(t)
	if key.N.BitLen() != 2048 {
		t.Errorf("unexpected key size %d", key.N.BitLen())
	}
}

func TestDKIMSignStructureAndVerify(t *testing.T) {
	key := mustKey(t)
	m := New().
		SetFrom("Ada Lovelace <ada@example.com>").
		AddTo("grace@example.com").
		SetSubject("DKIM test").
		SetText("Hello DKIM").
		SetDate(time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)).
		SetMessageID("dkim-1@example.com").
		SetBoundary("B")
	m.DKIM = &DKIM{
		Domain:      "example.com",
		Selector:    "mail",
		PrivateKey:  key,
		HeaderCanon: CanonRelaxed,
		BodyCanon:   CanonRelaxed,
		Time:        time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC),
	}
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.HasPrefix(s, "DKIM-Signature: ") {
		t.Fatalf("message does not start with DKIM-Signature:\n%s", s[:120])
	}
	line, _, _ := strings.Cut(s, crlf)
	tags := parseDKIMTags(line[len("DKIM-Signature: "):])
	for k, want := range map[string]string{
		"v": "1", "a": "rsa-sha256", "c": "relaxed/relaxed",
		"d": "example.com", "s": "mail", "t": "1767366245",
	} {
		if tags[k] != want {
			t.Errorf("tag %s = %q, want %q", k, tags[k], want)
		}
	}
	if tags["h"] != "From:To:Subject:Date:Message-ID:MIME-Version:Content-Type" {
		t.Errorf("h tag = %q", tags["h"])
	}
	if tags["bh"] == "" || tags["b"] == "" {
		t.Fatal("missing bh or b tag")
	}

	// Cryptographically verify the signature over the reconstructed, canonical
	// signed-header block. This asserts the signature is correct for the fixed
	// key + message.
	verifyDKIM(t, raw, line, key)

	// Deterministic: PKCS#1 v1.5 signing yields identical output for a fixed
	// key + message.
	raw2, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	if string(raw2) != s {
		t.Error("DKIM signing is not deterministic")
	}
}

// verifyDKIM reconstructs the canonical signed data from raw and confirms the b=
// signature verifies against the key's public part.
func verifyDKIM(t *testing.T, raw []byte, dkimLine string, key *rsa.PrivateKey) {
	t.Helper()
	tags := parseDKIMTags(dkimLine[len("DKIM-Signature: "):])
	sig, err := base64.StdEncoding.DecodeString(tags["b"])
	if err != nil {
		t.Fatalf("decode b: %v", err)
	}
	// Strip the DKIM-Signature line back off to recover the original message.
	orig := raw[len(dkimLine)+len(crlf):]
	headerBlock, _ := splitMessage(orig)
	fields := parseHeaderFields(headerBlock)

	var buf strings.Builder
	for _, name := range strings.Split(tags["h"], ":") {
		if hf, ok := findLastHeader(fields, name); ok {
			buf.WriteString(canonicalizeHeader(hf, CanonRelaxed))
			buf.WriteString(crlf)
		}
	}
	// Rebuild the DKIM-Signature header with an empty b= value.
	emptyB := dkimLine[:strings.LastIndex(dkimLine, "b=")+2]
	buf.WriteString(canonicalizeHeader(headerField{
		name:  "DKIM-Signature",
		value: " " + emptyB[len("DKIM-Signature: "):],
	}, CanonRelaxed))

	hashed := sha256.Sum256([]byte(buf.String()))
	if err := rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, hashed[:], sig); err != nil {
		t.Errorf("DKIM signature failed to verify: %v", err)
	}
}

func TestDKIMSimpleCanon(t *testing.T) {
	key := mustKey(t)
	m := newFixed().SetSubject("Simple").SetText("body").
		WithDKIM(&DKIM{
			Domain: "example.com", Selector: "s", PrivateKey: key,
			HeaderCanon: CanonSimple, BodyCanon: CanonSimple,
			Time: time.Unix(1000, 0),
		})
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	line, _, _ := strings.Cut(string(raw), crlf)
	if tags := parseDKIMTags(line[len("DKIM-Signature: "):]); tags["c"] != "simple/simple" {
		t.Errorf("c = %q, want simple/simple", tags["c"])
	}
}

func TestDKIMConfigError(t *testing.T) {
	d := &DKIM{Domain: "example.com"} // missing selector + key
	if _, err := d.Sign([]byte("From: a@b.com\r\n\r\nhi")); err == nil {
		t.Error("expected config error")
	}
	m := newFixed().SetText("x").WithDKIM(&DKIM{Domain: "x"})
	if _, err := m.Build(); err == nil {
		t.Error("expected build to fail with incomplete DKIM")
	}
}

func TestCanonicalizeBody(t *testing.T) {
	// Relaxed strips trailing whitespace and collapses internal runs, keeping a
	// single terminating CRLF.
	got := string(canonicalizeBody([]byte("a  b \r\n\r\n\r\n"), CanonRelaxed))
	if got != "a b"+crlf {
		t.Errorf("relaxed body = %q", got)
	}
	// Empty body canonicalizes to a single CRLF.
	if got := string(canonicalizeBody(nil, CanonRelaxed)); got != crlf {
		t.Errorf("empty body = %q", got)
	}
	// Simple keeps interior content but trims trailing empty lines.
	if got := string(canonicalizeBody([]byte("a  b\r\n\r\n"), CanonSimple)); got != "a  b"+crlf {
		t.Errorf("simple body = %q", got)
	}
}

func TestUnfoldAndCollapse(t *testing.T) {
	if got := unfoldAndCollapse("  a \r\n  b\tc  "); got != "a b c" {
		t.Errorf("unfoldAndCollapse = %q", got)
	}
}

// parseDKIMTags splits a DKIM-Signature value into its tag map.
func parseDKIMTags(v string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(v, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if eq := strings.IndexByte(part, '='); eq >= 0 {
			out[strings.TrimSpace(part[:eq])] = strings.TrimSpace(part[eq+1:])
		}
	}
	return out
}
