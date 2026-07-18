package nodemailer

import "testing"

func TestAddressLocalDomain(t *testing.T) {
	a := Address{Address: "Ada@Example.COM"}
	if a.Local() != "Ada" {
		t.Errorf("Local = %q", a.Local())
	}
	if a.Domain() != "Example.COM" {
		t.Errorf("Domain = %q", a.Domain())
	}
	empty := Address{Address: "invalid"}
	if empty.Local() != "" || empty.Domain() != "" {
		t.Errorf("expected empty for address without @")
	}
}

func TestAddressEqual(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"ada@example.com", "ada@EXAMPLE.com", true},
		{"ada@example.com", "ADA@example.com", false},
		{"a@x.com", "b@x.com", false},
	}
	for _, c := range cases {
		a := Address{Address: c.a}
		b := Address{Address: c.b}
		if got := a.Equal(b); got != c.want {
			t.Errorf("Equal(%q,%q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestNormalizeAddress(t *testing.T) {
	got, err := NormalizeAddress("Ada Lovelace <Ada@Example.COM>")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Ada@example.com" {
		t.Errorf("NormalizeAddress = %q", got)
	}
	if _, err := NormalizeAddress("not-an-address"); err == nil {
		t.Error("expected error for invalid address")
	}
}

func TestFormatAddressList(t *testing.T) {
	addrs := []Address{
		{Name: "Ada Lovelace", Address: "ada@example.com"},
		{Address: "grace@example.com"},
	}
	got := FormatAddressList(addrs)
	want := `"Ada Lovelace" <ada@example.com>, <grace@example.com>`
	if got != want {
		t.Errorf("FormatAddressList = %q, want %q", got, want)
	}
}
