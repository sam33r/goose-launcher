package markup

import (
	"image/color"
	"strings"
	"testing"
)

func TestParse_PlainText(t *testing.T) {
	plain, spans, err := Parse("hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plain != "hello world" {
		t.Errorf("plain = %q, want %q", plain, "hello world")
	}
	if len(spans) != 1 || spans[0].Text != "hello world" {
		t.Fatalf("spans = %+v, want one unstyled span", spans)
	}
	if spans[0].Bold || spans[0].Italic || spans[0].Underline || spans[0].FG != nil || spans[0].BG != nil {
		t.Errorf("plain span should carry no styling, got %+v", spans[0])
	}
}

func TestParse_Empty(t *testing.T) {
	plain, spans, err := Parse("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plain != "" {
		t.Errorf("plain = %q, want empty", plain)
	}
	if len(spans) != 0 {
		t.Errorf("spans = %+v, want none", spans)
	}
}

func TestParse_BoldItalic(t *testing.T) {
	plain, spans, err := Parse("<b>hello</b> <i>world</i>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plain != "hello world" {
		t.Errorf("plain = %q, want %q", plain, "hello world")
	}
	if len(spans) != 3 {
		t.Fatalf("expected 3 spans (bold, space, italic), got %d: %+v", len(spans), spans)
	}
	if !spans[0].Bold || spans[0].Text != "hello" {
		t.Errorf("span[0] = %+v, want bold 'hello'", spans[0])
	}
	if spans[1].Bold || spans[1].Italic || spans[1].Text != " " {
		t.Errorf("span[1] = %+v, want unstyled ' '", spans[1])
	}
	if !spans[2].Italic || spans[2].Text != "world" {
		t.Errorf("span[2] = %+v, want italic 'world'", spans[2])
	}
}

func TestParse_Nested(t *testing.T) {
	_, spans, err := Parse("<b>bold <i>bi</i> bold</b>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spans) != 3 {
		t.Fatalf("want 3 spans, got %d: %+v", len(spans), spans)
	}
	if !(spans[0].Bold && !spans[0].Italic && spans[0].Text == "bold ") {
		t.Errorf("span[0] = %+v", spans[0])
	}
	if !(spans[1].Bold && spans[1].Italic && spans[1].Text == "bi") {
		t.Errorf("span[1] = %+v", spans[1])
	}
	if !(spans[2].Bold && !spans[2].Italic && spans[2].Text == " bold") {
		t.Errorf("span[2] = %+v", spans[2])
	}
}

func TestParse_Adjacent(t *testing.T) {
	_, spans, err := Parse("<b>a</b><b>b</b>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Adjacent spans with identical style should merge.
	if len(spans) != 1 {
		t.Fatalf("want 1 merged span, got %d: %+v", len(spans), spans)
	}
	if !spans[0].Bold || spans[0].Text != "ab" {
		t.Errorf("merged span = %+v, want bold 'ab'", spans[0])
	}
}

func TestParse_SpanForegroundHex(t *testing.T) {
	_, spans, err := Parse(`<span foreground="#ff0000">red</span>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spans) != 1 || spans[0].FG == nil {
		t.Fatalf("want one span with FG set, got %+v", spans)
	}
	if *spans[0].FG != (color.NRGBA{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF}) {
		t.Errorf("FG = %+v, want pure red", *spans[0].FG)
	}
}

func TestParse_SpanShortHex(t *testing.T) {
	_, spans, err := Parse(`<span fg="#abc">x</span>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *spans[0].FG != (color.NRGBA{R: 0xAA, G: 0xBB, B: 0xCC, A: 0xFF}) {
		t.Errorf("short hex expansion wrong: %+v", *spans[0].FG)
	}
}

func TestParse_SpanNamedColor(t *testing.T) {
	_, spans, err := Parse(`<span foreground="red">x</span>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spans[0].FG == nil {
		t.Fatalf("FG not set for named color")
	}
}

func TestParse_SpanUnknownColor(t *testing.T) {
	_, _, err := Parse(`<span foreground="magentaish">x</span>`)
	if err == nil {
		t.Fatal("expected error for unknown color name")
	}
}

func TestParse_SpanBackgroundRoundtrip(t *testing.T) {
	// Background is parsed even though rendering is deferred (TODO(markup-bg)).
	_, spans, err := Parse(`<span background="#112233">x</span>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spans[0].BG == nil {
		t.Fatal("BG should round-trip through the parser")
	}
	if *spans[0].BG != (color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xFF}) {
		t.Errorf("BG = %+v", *spans[0].BG)
	}
}

func TestParse_UnderlineRoundtrip(t *testing.T) {
	// <u> is parsed; rendering is deferred (TODO(markup-underline)).
	_, spans, err := Parse("<u>x</u>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !spans[0].Underline {
		t.Fatal("Underline should round-trip through the parser")
	}
}

func TestParse_Entities(t *testing.T) {
	// encoding/xml decodes standard entities for us.
	plain, _, err := Parse("a &lt; b &amp; c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plain != "a < b & c" {
		t.Errorf("plain = %q, want %q", plain, "a < b & c")
	}
}

func TestParse_Malformed(t *testing.T) {
	cases := []string{
		"<unterminated",
		"</b>",        // end tag with nothing open
		"<b>unclosed", // never closed
		"<unknown>x</unknown>",
		"<b><i>mis</b></i>", // misnested
	}
	for _, in := range cases {
		if _, _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q): expected error, got nil", in)
		}
	}
}

func TestParse_SpanNoAttrs(t *testing.T) {
	// A <span> with no attributes is valid but has no effect on style.
	_, spans, err := Parse("<span>x</span>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spans) != 1 || spans[0].Text != "x" {
		t.Fatalf("spans = %+v", spans)
	}
	if spans[0].FG != nil || spans[0].BG != nil || spans[0].Bold || spans[0].Italic {
		t.Errorf("empty <span> should not add styling, got %+v", spans[0])
	}
}

func TestParse_SpansConcatEqualsPlain(t *testing.T) {
	// Invariant: joining span text reconstructs plain.
	inputs := []string{
		"hello",
		"<b>a</b>b<i>c</i>",
		"<b><i>nest</i>ed</b>",
		`pre <span fg="red">mid</span> post`,
	}
	for _, in := range inputs {
		plain, spans, err := Parse(in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", in, err)
		}
		var sb strings.Builder
		for _, s := range spans {
			sb.WriteString(s.Text)
		}
		if sb.String() != plain {
			t.Errorf("Parse(%q): spans concat = %q, plain = %q", in, sb.String(), plain)
		}
	}
}
