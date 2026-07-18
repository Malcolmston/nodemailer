package nodemailer

import "testing"

func TestHTMLToText(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello world", "hello world"},
		{"paragraphs", "<p>one</p><p>two</p>", "one\n\ntwo"},
		{"break", "a<br>b", "a\nb"},
		{"entities", "Tom &amp; Jerry &lt;3", "Tom & Jerry <3"},
		{"strip_tags", `<a href="x">link</a> text`, "link text"},
		{"script_removed", "keep<script>var x=1;</script>me", "keepme"},
		{"style_removed", "<style>p{color:red}</style>body", "body"},
		{"list", "<ul><li>a</li><li>b</li></ul>", "* a\n* b"},
		{"collapse_ws", "a   \t  b", "a b"},
		{"nested", "<div><p>Hi <b>there</b></p></div>", "Hi there"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := HTMLToText(c.in)
			if got != c.want {
				t.Errorf("HTMLToText(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestGenerateTextFromHTML(t *testing.T) {
	m := New().SetHTML("<p>Hello</p><p>World</p>")
	m.GenerateTextFromHTML()
	if m.Text != "Hello\n\nWorld" {
		t.Errorf("Text = %q", m.Text)
	}

	// Does not overwrite an existing text body.
	m2 := New().SetHTML("<p>x</p>").SetText("keep")
	m2.GenerateTextFromHTML()
	if m2.Text != "keep" {
		t.Errorf("Text overwritten: %q", m2.Text)
	}
}

func BenchmarkHTMLToText(b *testing.B) {
	html := "<html><body><h1>Title</h1><p>Some <b>bold</b> and <i>italic</i> text with &amp; entities.</p>" +
		"<ul><li>item one</li><li>item two</li></ul><script>ignore()</script></body></html>"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = HTMLToText(html)
	}
}
