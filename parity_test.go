package nodemailer

import (
	"reflect"
	"testing"
)

// The vectors in this file are transcribed verbatim from nodemailer's own test
// suite (github.com/nodemailer/nodemailer, lib+test):
//
//   - test/addressparser/addressparser-test.js
//   - test/qp/qp-test.js
//   - test/base64/base64-test.js
//   - test/mime-funcs/mime-funcs-test.js
//
// Each TestParity* function asserts the Go port reproduces the exact
// known-answer outputs the upstream library asserts.

// ---------------------------------------------------------------------------
// addressparser
// ---------------------------------------------------------------------------

func leaf(name, address string) ParsedAddr { return ParsedAddr{Name: name, Address: address} }

func group(name string, members ...ParsedAddr) ParsedAddr {
	return ParsedAddr{Name: name, IsGroup: true, Group: members}
}

func TestParityAddressparser(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []ParsedAddr
	}{
		{"single", "andris@tr.ee", []ParsedAddr{leaf("", "andris@tr.ee")}},
		{"multiple", "andris@tr.ee, andris@example.com", []ParsedAddr{leaf("", "andris@tr.ee"), leaf("", "andris@example.com")}},
		{"unquoted name", "andris <andris@tr.ee>", []ParsedAddr{leaf("andris", "andris@tr.ee")}},
		{"quoted name", `"reinman, andris" <andris@tr.ee>`, []ParsedAddr{leaf("reinman, andris", "andris@tr.ee")}},
		{"quoted semicolons", `"reinman; andris" <andris@tr.ee>`, []ParsedAddr{leaf("reinman; andris", "andris@tr.ee")}},
		{"unquoted name unquoted address", "andris andris@tr.ee", []ParsedAddr{leaf("andris", "andris@tr.ee")}},
		{"ipv6 literal colon", "user@[IPv6:2001:db8::1]", []ParsedAddr{leaf("", "user@[IPv6:2001:db8::1]")}},
		{"unclosed literal keeps recipients", "alice@example.com, bob[@example.com, carol@example.com", []ParsedAddr{leaf("", "alice@example.com"), leaf("", "bob[@example.com"), leaf("", "carol@example.com")}},
		{"comma after unclosed literal", "a@[b, c@d", []ParsedAddr{leaf("", "a@[b"), leaf("", "c@d")}},
		{"empty group", "Undisclosed:;", []ParsedAddr{group("Undisclosed")}},
		{"address group", "Disclosed:andris@tr.ee, andris@example.com;", []ParsedAddr{group("Disclosed", leaf("", "andris@tr.ee"), leaf("", "andris@example.com"))}},
		{"semicolon delimiter", "andris@tr.ee; andris@example.com;", []ParsedAddr{leaf("", "andris@tr.ee"), leaf("", "andris@example.com")}},
		{"name from comment", "andris@tr.ee (andris)", []ParsedAddr{leaf("andris", "andris@tr.ee")}},
		{"skip comment", "andris@tr.ee (reinman) andris", []ParsedAddr{leaf("andris", "andris@tr.ee")}},
		{"missing address", "andris", []ParsedAddr{leaf("andris", "")}},
		{"apostrophe in name", "O'Neill", []ParsedAddr{leaf("O'Neill", "")}},
		{"unescaped colon flattened group", "FirstName Surname-WithADash :: Company <firstname@company.com>", []ParsedAddr{group("FirstName Surname-WithADash", leaf("Company", "firstname@company.com"))}},
		{"invalid double-domain kept", "name@address.com@address2.com", []ParsedAddr{leaf("", "name@address.com@address2.com")}},
		{"unexpected angle", "reinman > andris < test <andris@tr.ee>", []ParsedAddr{leaf("reinman > andris", "andris@tr.ee")}},
		{"escapes", `"Firstname \" \\\, Lastname \(Test\)" test@example.com`, []ParsedAddr{leaf(`Firstname " \, Lastname (Test)`, "test@example.com")}},
		{"quoted username", `"test@subdomain.com"@example.com`, []ParsedAddr{leaf("", "test@subdomain.com@example.com")}},
		{"quoted local-part security", `"xclow3n@gmail.com x"@internal.domain`, []ParsedAddr{leaf("", "xclow3n@gmail.com x@internal.domain")}},
		{"quoted attacker domain", `"user@attacker.com"@legitimate.com`, []ParsedAddr{leaf("", "user@attacker.com@legitimate.com")}},
		{"multiple @ quoted", `"a@b@c"@example.com`, []ParsedAddr{leaf("", "a@b@c@example.com")}},
		{"escaped quote in quoted string", `"test\"quote"@example.com`, []ParsedAddr{leaf("", `test"quote@example.com`)}},
		{"escaped backslashes", `"test\\backslash"@example.com`, []ParsedAddr{leaf("", `test\backslash@example.com`)}},
		{"special local parts +", "user+tag@example.com", []ParsedAddr{leaf("", "user+tag@example.com")}},
		{"multiple consecutive delimiters", "a@example.com,,,b@example.com", []ParsedAddr{leaf("", "a@example.com"), leaf("", "b@example.com")}},
		{"mixed quotes and unquoted", `"quoted" unquoted@example.com`, []ParsedAddr{leaf("quoted", "unquoted@example.com")}},
		{"only name", "John Doe", []ParsedAddr{leaf("John Doe", "")}},
		{"unicode display name", "Jüri Õunapuu <juri@example.com>", []ParsedAddr{leaf("Jüri Õunapuu", "juri@example.com")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseAddresses(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ParseAddresses(%q)\n got = %#v\nwant = %#v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParityAddressparserQuotedAngle(t *testing.T) {
	// Quotes are preserved as part of the address when wrapped in <>.
	got := ParseAddresses(`Name <"user@domain.com"@example.com>`)
	if len(got) != 1 || got[0].Name != "Name" || got[0].Address != `"user@domain.com"@example.com` {
		t.Fatalf("got %#v", got)
	}
}

func TestParityAddressparserEmptyAndWhitespace(t *testing.T) {
	if got := ParseAddresses(""); len(got) != 0 {
		t.Fatalf("empty: got %#v", got)
	}
	if got := ParseAddresses("   "); len(got) != 0 {
		t.Fatalf("whitespace: got %#v", got)
	}
	if got := ParseAddresses("<>"); len(got) != 1 || got[0].Address != "" {
		t.Fatalf("empty angle: got %#v", got)
	}
}

func TestParityAddressparserFlatten(t *testing.T) {
	in := "Test User <test.user@mail.ee>, Disclosed:andris@tr.ee, andris@example.com;,,,, Undisclosed:; bob@example.com BOB;"
	got := ParseAddressesFlatten(in)
	want := []ParsedAddr{
		leaf("Test User", "test.user@mail.ee"),
		leaf("", "andris@tr.ee"),
		leaf("", "andris@example.com"),
		leaf("BOB", "bob@example.com"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("flatten\n got = %#v\nwant = %#v", got, want)
	}
}

func TestParityAddressparserMixedGroup(t *testing.T) {
	in := "Test User <test.user@mail.ee>, Disclosed:andris@tr.ee, andris@example.com;,,,, Undisclosed:;"
	got := ParseAddresses(in)
	want := []ParsedAddr{
		leaf("Test User", "test.user@mail.ee"),
		group("Disclosed", leaf("", "andris@tr.ee"), leaf("", "andris@example.com")),
		group("Undisclosed"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mixed group\n got = %#v\nwant = %#v", got, want)
	}
}

func TestParityAddressparserNestedGroups(t *testing.T) {
	// Deeply nested groups flatten to a single level.
	got := ParseAddresses("Outer:Inner:deep@example.com;;")
	if len(got) != 1 || !got[0].IsGroup || got[0].Name != "Outer" || len(got[0].Group) != 1 || got[0].Group[0].Address != "deep@example.com" {
		t.Fatalf("nested: got %#v", got)
	}
}

// ---------------------------------------------------------------------------
// quoted-printable
// ---------------------------------------------------------------------------

func TestParityQPEncode(t *testing.T) {
	cases := [][2]string{
		{"abcd= ÕÄÖÜ", "abcd=3D =C3=95=C3=84=C3=96=C3=9C"},
		{"foo bar  ", "foo bar =20"},
		{"foo bar\t\t", "foo bar\t=09"},
		{"foo \r\nbar", "foo=20\r\nbar"},
	}
	for _, c := range cases {
		if got := QPEncode([]byte(c[0])); got != c[1] {
			t.Errorf("QPEncode(%q) = %q, want %q", c[0], got, c[1])
		}
	}
	if got := QPEncode([]byte{0x00, 0x01, 0x02, 0x20, 0x03}); got != "=00=01=02 =03" {
		t.Errorf("QPEncode(bytes) = %q, want %q", got, "=00=01=02 =03")
	}
}

func TestParityQPWrap(t *testing.T) {
	cases := [][2]string{
		{"tere, tere, vana kere, kuidas sul l=C3=A4heb?", "tere, tere, vana =\r\nkere, kuidas sul =\r\nl=C3=A4heb?"},
		{"=C3=A4=C3=A4=C3=A4=C3=A4=C3=A4=C3=A4=C3=A4=C3=A4=C3=A4=C3=A4", "=C3=A4=C3=A4=\r\n=C3=A4=C3=A4=\r\n=C3=A4=C3=A4=\r\n=C3=A4=C3=A4=\r\n=C3=A4=C3=A4"},
		{"1234567890123456789=C3=A40", "1234567890123456789=\r\n=C3=A40"},
		{"123456789012345678  90", "123456789012345678 =\r\n 90"},
	}
	for _, c := range cases {
		if got := QPWrap(c[0], 20); got != c[1] {
			t.Errorf("QPWrap(%q, 20)\n got = %q\nwant = %q", c[0], got, c[1])
		}
	}
	if got := QPWrap("alfa palfa kalfa ralfa\r", 10); got != "alfa palf=\r\na kalfa =\r\nralfa\r" {
		t.Errorf("QPWrap CR = %q", got)
	}
}

// ---------------------------------------------------------------------------
// base64
// ---------------------------------------------------------------------------

func TestParityBase64Encode(t *testing.T) {
	cases := [][2]string{
		{"abcd= ÕÄÖÜ", "YWJjZD0gw5XDhMOWw5w="},
		{"foo bar  ", "Zm9vIGJhciAg"},
		{"foo bar\t\t", "Zm9vIGJhcgkJ"},
		{"foo \r\nbar", "Zm9vIA0KYmFy"},
	}
	for _, c := range cases {
		if got := Base64Encode([]byte(c[0])); got != c[1] {
			t.Errorf("Base64Encode(%q) = %q, want %q", c[0], got, c[1])
		}
	}
	if got := Base64Encode([]byte{0x00, 0x01, 0x02, 0x20, 0x03}); got != "AAECIAM=" {
		t.Errorf("Base64Encode(bytes) = %q", got)
	}
}

func TestParityBase64Wrap(t *testing.T) {
	in := "dGVyZSwgdGVyZSwgdmFuYSBrZXJlLCBrdWlkYXMgc3VsIGzDpGhlYj8="
	want := "dGVyZSwgdGVyZSwgdmFu\r\nYSBrZXJlLCBrdWlkYXMg\r\nc3VsIGzDpGhlYj8="
	if got := Base64Wrap(in, 20); got != want {
		t.Errorf("Base64Wrap\n got = %q\nwant = %q", got, want)
	}
	// Exact multiple of lineLength must not gain a trailing CRLF.
	exact := repeat("A", 152)
	if got := Base64Wrap(exact, 76); got != repeat("A", 76)+"\r\n"+repeat("A", 76) {
		t.Errorf("Base64Wrap exact multiple = %q", got)
	}
	// Content preserved and no trailing CR/LF across chunk boundaries.
	for _, n := range []int{1, 7, 8, 9, 15, 16, 17, 8191, 8192, 8193, 16383, 16384, 16385} {
		input := repeat("A", n)
		wrapped := Base64Wrap(input, 8)
		if l := len(wrapped); l > 0 && (wrapped[l-1] == '\r' || wrapped[l-1] == '\n') {
			t.Errorf("size %d: output ends in CR/LF", n)
		}
		if joinNoCRLF(wrapped) != input {
			t.Errorf("size %d: content not preserved", n)
		}
	}
}

// ---------------------------------------------------------------------------
// mime-funcs
// ---------------------------------------------------------------------------

func TestParityIsPlainText(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"abc", true},
		{"abc\x02", false},
		{"abcõ", false},
		{"az09\t\r\n~!?", true},
		{"az09\n\x08!?", false},
		{"az09\nõ!?", false},
	}
	for _, c := range cases {
		if got := IsPlainText(c.in); got != c.want {
			t.Errorf("IsPlainText(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParityHasLongerLines(t *testing.T) {
	if HasLongerLines("abc\ndef", 5) {
		t.Error("expected no longer lines")
	}
	if !HasLongerLines("juf\nabcdef\nghi", 5) {
		t.Error("expected a longer line")
	}
}

func TestParityEncodeWord(t *testing.T) {
	if got := EncodeMimeWord([]byte("See on õhin test"), "Q"); got != "=?UTF-8?Q?See_on_=C3=B5hin_test?=" {
		t.Errorf("EncodeMimeWord Q = %q", got)
	}
	if got := EncodeMimeWord([]byte("See on õhin test"), "B"); got != "=?UTF-8?B?U2VlIG9uIMO1aGluIHRlc3Q=?=" {
		t.Errorf("EncodeMimeWord B = %q", got)
	}
}

// helpers

func repeat(s string, n int) string {
	b := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		b = append(b, s...)
	}
	return string(b)
}

func joinNoCRLF(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\r' || s[i] == '\n' {
			continue
		}
		b = append(b, s[i])
	}
	return string(b)
}
