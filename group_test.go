package nodemailer

import (
	"strings"
	"testing"
)

func TestParseAddressGroup(t *testing.T) {
	g, err := ParseAddressGroup("Friends: ada@example.com, Grace <grace@example.com>;")
	if err != nil {
		t.Fatal(err)
	}
	if g.Name != "Friends" {
		t.Errorf("name = %q", g.Name)
	}
	if len(g.Addresses) != 2 {
		t.Fatalf("got %d members, want 2", len(g.Addresses))
	}
	if g.Addresses[0].Address != "ada@example.com" || g.Addresses[1].Name != "Grace" {
		t.Errorf("unexpected members: %+v", g.Addresses)
	}
}

func TestParseAddressGroupEmpty(t *testing.T) {
	g, err := ParseAddressGroup("Undisclosed recipients:;")
	if err != nil {
		t.Fatal(err)
	}
	if g.Name != "Undisclosed recipients" || len(g.Addresses) != 0 {
		t.Errorf("unexpected empty group: %+v", g)
	}
	if got := g.String(); got != "Undisclosed recipients: ;" {
		t.Errorf("String = %q", got)
	}
}

func TestParseAddressGroupErrors(t *testing.T) {
	if _, err := ParseAddressGroup("no colon here"); err == nil {
		t.Error("expected error for missing colon")
	}
	if _, err := ParseAddressGroup("Bad: not@an@address;"); err == nil {
		t.Error("expected error for invalid member")
	}
}

func TestAddressGroupString(t *testing.T) {
	g := AddressGroup{Name: "Team", Addresses: []Address{
		{Address: "a@x.com"}, {Name: "B", Address: "b@y.com"},
	}}
	want := `Team: <a@x.com>, "B" <b@y.com>;`
	if got := g.String(); got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
}

func TestMessageGroupHeaders(t *testing.T) {
	m := newFixed().
		SetSubject("Groups").
		SetText("hi").
		AddToGroup("Team", "ada@example.com", "grace@example.com").
		AddCcGroup("Board", "carl@example.com")
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "Team: ") || !strings.Contains(s, "ada@example.com") {
		t.Errorf("To group not rendered:\n%s", s)
	}
	if !strings.Contains(s, "Board: <carl@example.com>;") {
		t.Errorf("Cc group not rendered:\n%s", s)
	}
	// Group members must appear as envelope recipients.
	rcpts := m.Recipients()
	joined := strings.Join(rcpts, ",")
	for _, want := range []string{"ada@example.com", "grace@example.com", "carl@example.com"} {
		if !strings.Contains(joined, want) {
			t.Errorf("recipient %q missing from %v", want, rcpts)
		}
	}
}

func TestGroupOnlyRecipient(t *testing.T) {
	// A message whose only recipients are in a group must still build.
	m := New().SetFrom("ada@example.com").SetText("hi").
		AddToGroup("Team", "grace@example.com")
	if _, err := m.Build(); err != nil {
		t.Fatalf("group-only message failed to build: %v", err)
	}
}

func TestAddGroupParseError(t *testing.T) {
	m := New().SetFrom("ada@example.com").AddToGroup("Bad", "not-an-address")
	if m.Err() == nil {
		t.Error("expected deferred parse error")
	}
}
