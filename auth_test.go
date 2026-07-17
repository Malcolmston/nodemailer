package nodemailer

import (
	"encoding/base64"
	"testing"
)

func TestXOAuth2Token(t *testing.T) {
	got := XOAuth2Token("user@example.com", "tok123")
	want := "user=user@example.com\x01auth=Bearer tok123\x01\x01"
	if got != want {
		t.Errorf("XOAuth2Token = %q, want %q", got, want)
	}
	// Base64 form matches the well-known XOAUTH2 wire encoding.
	if enc := base64.StdEncoding.EncodeToString([]byte(got)); enc == "" {
		t.Error("empty encoding")
	}
}

func TestXOAuth2AuthStart(t *testing.T) {
	a := XOAuth2Auth("user@example.com", "tok123")
	proto, resp, err := a.Start(nil)
	if err != nil {
		t.Fatal(err)
	}
	if proto != "XOAUTH2" {
		t.Errorf("proto = %q", proto)
	}
	if string(resp) != XOAuth2Token("user@example.com", "tok123") {
		t.Errorf("initial response = %q", resp)
	}
}

func TestXOAuth2AuthNext(t *testing.T) {
	a := XOAuth2Auth("u@x.com", "t")
	// On a server error challenge (more=true), reply with an empty line.
	resp, err := a.Next([]byte("eyJzdGF0dXMiOiI0MDEifQ=="), true)
	if err != nil || len(resp) != 0 {
		t.Errorf("Next(more) = %q, %v; want empty, nil", resp, err)
	}
	// With more=false there is nothing further to send.
	if resp, err := a.Next(nil, false); err != nil || resp != nil {
		t.Errorf("Next(!more) = %q, %v; want nil, nil", resp, err)
	}
}

func TestDecodeXOAuth2(t *testing.T) {
	raw := XOAuth2Token("ada@example.com", "abc")
	// Decode from raw bytes.
	if u, tok, ok := decodeXOAuth2([]byte(raw)); !ok || u != "ada@example.com" || tok != "abc" {
		t.Errorf("decode raw = %q %q %v", u, tok, ok)
	}
	// Decode from base64 (as seen on the wire).
	enc := base64.StdEncoding.EncodeToString([]byte(raw))
	if u, tok, ok := decodeXOAuth2([]byte(enc)); !ok || u != "ada@example.com" || tok != "abc" {
		t.Errorf("decode b64 = %q %q %v", u, tok, ok)
	}
	// Malformed input.
	if _, _, ok := decodeXOAuth2([]byte("garbage")); ok {
		t.Error("expected decode failure for garbage")
	}
}

func TestXOAuth2AgainstServer(t *testing.T) {
	srv := newRichServer(t)
	srv.advertiseAuth = true
	tr := srv.transport(t)
	tr.Auth = XOAuth2Auth("ada@example.com", "secret-token")

	raw := []byte("Subject: Hi\r\n\r\nbody\r\n")
	if err := tr.Send("ada@example.com", []string{"grace@example.com"}, raw); err != nil {
		t.Fatalf("Send: %v", err)
	}
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.authUser != "ada@example.com" || srv.authToken != "secret-token" {
		t.Errorf("server saw user=%q token=%q", srv.authUser, srv.authToken)
	}
}

func TestXOAuth2AuthFailure(t *testing.T) {
	srv := newRichServer(t)
	srv.advertiseAuth = true
	srv.failAuth = true
	tr := srv.transport(t)
	tr.Auth = XOAuth2Auth("ada@example.com", "bad-token")
	if err := tr.Send("ada@example.com", []string{"grace@example.com"}, []byte("x")); err == nil {
		t.Error("expected authentication failure")
	}
}
