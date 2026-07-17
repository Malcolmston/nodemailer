package nodemailer

import (
	"strings"
	"testing"
)

func TestPriorityHeaders(t *testing.T) {
	raw, err := newFixed().SetSubject("s").SetText("x").SetPriority(PriorityHigh).Build()
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, want := range []string{"X-Priority: 1 (Highest)", "X-MSMail-Priority: High", "Importance: High"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q", want)
		}
	}

	raw, _ = newFixed().SetText("x").SetPriority(PriorityLow).Build()
	if !strings.Contains(string(raw), "Importance: Low") {
		t.Error("missing low importance")
	}

	// Normal priority emits no priority headers.
	raw, _ = newFixed().SetText("x").SetPriority(PriorityNormal).Build()
	if strings.Contains(string(raw), "X-Priority") {
		t.Error("normal priority should not emit headers")
	}
}

func TestThreadingHeaders(t *testing.T) {
	m := newFixed().SetSubject("Re: x").SetText("body").
		SetInReplyTo("<parent@example.com>").
		AddReferences("root@example.com", "<parent@example.com>")
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "In-Reply-To: <parent@example.com>") {
		t.Errorf("In-Reply-To missing:\n%s", s)
	}
	if !strings.Contains(s, "References: <root@example.com> <parent@example.com>") {
		t.Errorf("References missing/wrong:\n%s", s)
	}
}

func TestListHeaders(t *testing.T) {
	m := newFixed().SetText("x").
		SetListUnsubscribe("mailto:unsub@example.com", "https://example.com/unsub").
		SetListUnsubscribePost().
		AddListHeader("ID", "<newsletter.example.com>").
		AddListHeader("List-Help", "<mailto:help@example.com>")
	raw, err := m.Build()
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "List-Unsubscribe: <mailto:unsub@example.com>, <https://example.com/unsub>") {
		t.Errorf("List-Unsubscribe missing/wrong:\n%s", s)
	}
	if !strings.Contains(s, "List-Unsubscribe-Post: List-Unsubscribe=One-Click") {
		t.Error("List-Unsubscribe-Post missing")
	}
	if !strings.Contains(s, "List-ID: <newsletter.example.com>") {
		t.Error("List-ID missing (prefix should be added)")
	}
	if !strings.Contains(s, "List-Help: <mailto:help@example.com>") {
		t.Error("List-Help missing (already-prefixed key)")
	}
	if strings.Contains(s, "List-List-Help") {
		t.Error("double List- prefix applied")
	}
}
