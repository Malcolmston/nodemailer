package nodemailer

import (
	"strings"
	"testing"
)

func TestEncodeWord(t *testing.T) {
	if got := EncodeWord("utf-8", "plain ascii"); got != "plain ascii" {
		t.Errorf("ascii should be unchanged, got %q", got)
	}
	got := EncodeWord("utf-8", "Grüße")
	if !strings.HasPrefix(got, "=?utf-8?q?") && !strings.HasPrefix(got, "=?UTF-8?q?") {
		t.Errorf("expected encoded-word, got %q", got)
	}
	// Round-trips back through the decoder.
	back, err := DecodeHeaderWord(got)
	if err != nil {
		t.Fatal(err)
	}
	if back != "Grüße" {
		t.Errorf("round trip = %q", back)
	}
}

func TestDecodeHeaderWord(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain text", "plain text"},
		{"=?utf-8?q?Gr=C3=BC=C3=9Fe?=", "Grüße"},
		{"=?UTF-8?B?SGVsbG8=?=", "Hello"},
		{"=?ISO-8859-1?Q?caf=E9?=", "café"},
	}
	for _, c := range cases {
		got, err := DecodeHeaderWord(c.in)
		if err != nil {
			t.Errorf("DecodeHeaderWord(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("DecodeHeaderWord(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestGenerateMessageID(t *testing.T) {
	id := GenerateMessageID("example.com")
	if !strings.HasSuffix(id, "@example.com") {
		t.Errorf("id = %q, want @example.com suffix", id)
	}
	if strings.Contains(id, "<") || strings.Contains(id, ">") {
		t.Errorf("id should not carry angle brackets: %q", id)
	}
	// Uniqueness across calls.
	if GenerateMessageID("example.com") == id {
		t.Error("two generated ids collided")
	}
	// Empty domain falls back to localhost.
	if !strings.HasSuffix(GenerateMessageID(""), "@localhost") {
		t.Error("empty domain should default to localhost")
	}
}
