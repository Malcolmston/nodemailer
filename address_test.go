package nodemailer

import (
	"errors"
	"testing"
)

func TestParseAddress(t *testing.T) {
	cases := []struct {
		in       string
		wantName string
		wantAddr string
	}{
		{"ada@example.com", "", "ada@example.com"},
		{"Ada Lovelace <ada@example.com>", "Ada Lovelace", "ada@example.com"},
		{"  <grace@example.com>  ", "", "grace@example.com"},
		{`"Hopper, Grace" <grace@example.com>`, "Hopper, Grace", "grace@example.com"},
	}
	for _, c := range cases {
		got, err := ParseAddress(c.in)
		if err != nil {
			t.Fatalf("ParseAddress(%q) error: %v", c.in, err)
		}
		if got.Name != c.wantName || got.Address != c.wantAddr {
			t.Errorf("ParseAddress(%q) = %+v, want name=%q addr=%q", c.in, got, c.wantName, c.wantAddr)
		}
	}
}

func TestParseAddressInvalid(t *testing.T) {
	invalid := []string{
		"",
		"not-an-address",
		"missing@",
		"@example.com",
		"a@b",            // no dot in domain
		"a@@example.com", // double @ is rejected by parser
	}
	for _, in := range invalid {
		if _, err := ParseAddress(in); err == nil {
			t.Errorf("ParseAddress(%q) expected error, got nil", in)
		} else if !errors.Is(err, ErrInvalidAddress) {
			t.Errorf("ParseAddress(%q) error %v, want ErrInvalidAddress", in, err)
		}
	}
}

func TestParseAddressList(t *testing.T) {
	list, err := ParseAddressList("Ada <ada@example.com>, grace@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d addresses, want 2", len(list))
	}
	if list[0].Name != "Ada" || list[1].Address != "grace@example.com" {
		t.Errorf("unexpected list: %+v", list)
	}

	if empty, err := ParseAddressList("   "); err != nil || empty != nil {
		t.Errorf("empty list = %+v, %v; want nil, nil", empty, err)
	}

	if _, err := ParseAddressList("ada@example.com, bad@@x.com"); err == nil {
		t.Error("expected error for invalid list member")
	}
}

func TestAddressValidate(t *testing.T) {
	valid := []Address{
		{Address: "a@b.com"},
		{Name: "X", Address: "x.y+tag@sub.example.org"},
	}
	for _, a := range valid {
		if err := a.Validate(); err != nil {
			t.Errorf("Validate(%+v) = %v, want nil", a, err)
		}
	}
	invalid := []Address{
		{Address: ""},
		{Address: "nodomain"},
		{Address: "a@localhost"},
		{Address: "a b@x.com"},
		{Address: "a@.com"},
		{Address: "a@com."},
	}
	for _, a := range invalid {
		if err := a.Validate(); err == nil {
			t.Errorf("Validate(%+v) = nil, want error", a)
		}
	}
}

func TestAddressString(t *testing.T) {
	if got := (Address{Address: "a@b.com"}).String(); got != "<a@b.com>" {
		t.Errorf("plain address String = %q", got)
	}
	got := (Address{Name: "Björk", Address: "bjork@example.is"}).String()
	// Non-ASCII names must be RFC 2047 encoded.
	if got == "Björk <bjork@example.is>" {
		t.Errorf("expected encoded-word name, got raw %q", got)
	}
}
