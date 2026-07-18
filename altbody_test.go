package nodemailer

import (
	"bytes"
	"testing"
)

func TestSetAMPAndWatchHTML(t *testing.T) {
	m := New().SetAMP("<amp></amp>").SetWatchHTML("<watch>")
	if len(m.Alternatives) != 2 {
		t.Fatalf("got %d alternatives, want 2", len(m.Alternatives))
	}
	if m.Alternatives[0].ContentType != "text/x-amp-html" || m.Alternatives[0].Content != "<amp></amp>" {
		t.Errorf("amp alternative = %+v", m.Alternatives[0])
	}
	if m.Alternatives[1].ContentType != "text/watch-html" || m.Alternatives[1].Content != "<watch>" {
		t.Errorf("watch alternative = %+v", m.Alternatives[1])
	}
}

func TestSetAMPBuildsIntoMIME(t *testing.T) {
	raw, err := newFixed().
		SetSubject("Hi").
		SetText("t").
		SetHTML("<p>h</p>").
		SetAMP("<amp></amp>").
		Build()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("text/x-amp-html")) {
		t.Errorf("amp content type missing from output:\n%s", raw)
	}
}
