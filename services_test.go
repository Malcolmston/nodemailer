package nodemailer

import "testing"

func TestWellKnownService(t *testing.T) {
	cases := []struct {
		in     string
		host   string
		port   int
		secure bool
		ok     bool
	}{
		{"gmail", "smtp.gmail.com", 465, true, true},
		{"Gmail", "smtp.gmail.com", 465, true, true},
		{" GoogleMail ", "smtp.gmail.com", 465, true, true},
		{"hotmail", "smtp-mail.outlook.com", 587, false, true},
		{"sendgrid", "smtp.sendgrid.net", 587, false, true},
		{"yahoo", "smtp.mail.yahoo.com", 465, true, true},
		{"nope", "", 0, false, false},
	}
	for _, c := range cases {
		svc, ok := WellKnownService(c.in)
		if ok != c.ok {
			t.Errorf("WellKnownService(%q) ok=%v want %v", c.in, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if svc.Host != c.host || svc.Port != c.port || svc.Secure != c.secure {
			t.Errorf("WellKnownService(%q) = %+v, want host=%s port=%d secure=%v", c.in, svc, c.host, c.port, c.secure)
		}
	}
}

func TestWellKnownServiceNamesSorted(t *testing.T) {
	names := WellKnownServiceNames()
	if len(names) == 0 {
		t.Fatal("expected some service names")
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Fatalf("names not sorted: %q > %q", names[i-1], names[i])
		}
	}
	// Ensure no duplicate canonical names (aliases collapsed).
	seen := map[string]bool{}
	for _, n := range names {
		if seen[n] {
			t.Fatalf("duplicate service name %q", n)
		}
		seen[n] = true
	}
}

func TestNewServiceSMTP(t *testing.T) {
	tr, err := NewServiceSMTP("Gmail", "ada", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if tr.Host != "smtp.gmail.com" || tr.Port != 465 || !tr.TLS || tr.STARTTLS {
		t.Errorf("gmail transport = %+v", tr)
	}
	if tr.Username != "ada" || tr.Password != "secret" {
		t.Errorf("credentials not set: %+v", tr)
	}

	tr2, err := NewServiceSMTP("hotmail", "u", "p")
	if err != nil {
		t.Fatal(err)
	}
	if tr2.STARTTLS != true || tr2.TLS {
		t.Errorf("hotmail should use STARTTLS: %+v", tr2)
	}

	if _, err := NewServiceSMTP("unknown", "u", "p"); err == nil {
		t.Error("expected error for unknown service")
	}
}
