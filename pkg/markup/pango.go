// Package markup parses a small Pango-markup subset into styled text spans.
//
// We support the tags goose-launcher currently renders (<b>, <i>, fg color)
// plus a couple we parse but don't render yet (<u>, bg color). Keeping the
// deferred attributes in the grammar means producers can emit them today and
// get the visuals for free when rendering lands.
package markup

import (
	"encoding/xml"
	"fmt"
	"image/color"
	"io"
	"strconv"
	"strings"
)

// Span is a contiguous styled run of plain text.
type Span struct {
	Text      string
	Bold      bool
	Italic    bool
	Underline bool // parsed; rendering is TODO(markup-underline)
	FG        *color.NRGBA
	BG        *color.NRGBA // parsed; rendering is TODO(markup-bg)
}

// Parse returns the plain text (with tags stripped and XML entities decoded)
// and the list of styled spans covering it. Spans' concatenated Text equals
// plain. On any parse error the caller gets an error and should fall back to
// treating the input as literal text.
func Parse(s string) (plain string, spans []Span, err error) {
	// Wrap in a synthetic root so encoding/xml sees a well-formed document.
	dec := xml.NewDecoder(strings.NewReader("<r>" + s + "</r>"))
	dec.Strict = true

	var (
		stack    []Span // active style context; top-of-stack wins
		plainBuf strings.Builder
	)

	for {
		tok, e := dec.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return "", nil, e
		}

		switch t := tok.(type) {
		case xml.StartElement:
			name := strings.ToLower(t.Name.Local)
			if name == "r" {
				continue
			}
			style, e := startStyle(name, t.Attr, currentStyle(stack))
			if e != nil {
				return "", nil, e
			}
			stack = append(stack, style)

		case xml.EndElement:
			name := strings.ToLower(t.Name.Local)
			if name == "r" {
				continue
			}
			if len(stack) == 0 {
				return "", nil, fmt.Errorf("markup: unexpected </%s>", name)
			}
			stack = stack[:len(stack)-1]

		case xml.CharData:
			if len(t) == 0 {
				continue
			}
			plainBuf.Write(t)
			spans = appendSpan(spans, string(t), currentStyle(stack))
		}
	}

	if len(stack) != 0 {
		return "", nil, fmt.Errorf("markup: unclosed tag")
	}

	return plainBuf.String(), spans, nil
}

// currentStyle returns the active style (top of stack), or a zero Span.
func currentStyle(stack []Span) Span {
	if len(stack) == 0 {
		return Span{}
	}
	return stack[len(stack)-1]
}

// startStyle layers a new tag's attributes on top of the inherited style.
func startStyle(name string, attrs []xml.Attr, parent Span) (Span, error) {
	s := parent
	s.Text = "" // Text is per-chunk, not carried through the stack

	switch name {
	case "b":
		s.Bold = true
	case "i":
		s.Italic = true
	case "u":
		s.Underline = true
	case "span":
		for _, a := range attrs {
			key := strings.ToLower(a.Name.Local)
			switch key {
			case "foreground", "fgcolor", "fg", "color":
				c, err := parseColor(a.Value)
				if err != nil {
					return Span{}, fmt.Errorf("markup: span %s=%q: %w", key, a.Value, err)
				}
				s.FG = &c
			case "background", "bgcolor", "bg":
				c, err := parseColor(a.Value)
				if err != nil {
					return Span{}, fmt.Errorf("markup: span %s=%q: %w", key, a.Value, err)
				}
				s.BG = &c
			default:
				return Span{}, fmt.Errorf("markup: unsupported span attribute %q", a.Name.Local)
			}
		}
	default:
		return Span{}, fmt.Errorf("markup: unsupported tag <%s>", name)
	}
	return s, nil
}

// appendSpan adds a chunk of text under a given style. If the previous span
// has the same style, the text is merged into it.
func appendSpan(spans []Span, text string, style Span) []Span {
	style.Text = text
	if n := len(spans); n > 0 && sameStyle(spans[n-1], style) {
		spans[n-1].Text += text
		return spans
	}
	return append(spans, style)
}

func sameStyle(a, b Span) bool {
	return a.Bold == b.Bold &&
		a.Italic == b.Italic &&
		a.Underline == b.Underline &&
		colorEq(a.FG, b.FG) &&
		colorEq(a.BG, b.BG)
}

func colorEq(a, b *color.NRGBA) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return *a == *b
	}
}

// named is a short CSS-subset color map. Case-insensitive lookup via ToLower.
var named = map[string]color.NRGBA{
	"black":        {R: 0x00, G: 0x00, B: 0x00, A: 0xFF},
	"white":        {R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
	"red":          {R: 0xCC, G: 0x33, B: 0x33, A: 0xFF},
	"green":        {R: 0x33, G: 0xCC, B: 0x33, A: 0xFF},
	"blue":         {R: 0x33, G: 0x66, B: 0xCC, A: 0xFF},
	"yellow":       {R: 0xE5, G: 0xC0, B: 0x7B, A: 0xFF},
	"cyan":         {R: 0x4E, G: 0xC9, B: 0xB0, A: 0xFF},
	"magenta":      {R: 0xC5, G: 0x78, B: 0xDD, A: 0xFF},
	"gray":         {R: 0x88, G: 0x88, B: 0x88, A: 0xFF},
	"grey":         {R: 0x88, G: 0x88, B: 0x88, A: 0xFF},
	"lightred":     {R: 0xFF, G: 0x66, B: 0x66, A: 0xFF},
	"lightgreen":   {R: 0x66, G: 0xFF, B: 0x66, A: 0xFF},
	"lightblue":    {R: 0x66, G: 0x99, B: 0xFF, A: 0xFF},
	"lightyellow":  {R: 0xFF, G: 0xEE, B: 0x99, A: 0xFF},
	"lightcyan":    {R: 0x99, G: 0xEE, B: 0xEE, A: 0xFF},
	"lightmagenta": {R: 0xEE, G: 0x99, B: 0xEE, A: 0xFF},
	"darkred":      {R: 0x66, G: 0x00, B: 0x00, A: 0xFF},
	"darkgreen":    {R: 0x00, G: 0x66, B: 0x00, A: 0xFF},
	"darkblue":     {R: 0x00, G: 0x00, B: 0x66, A: 0xFF},
}

// parseColor accepts #rgb, #rrggbb, or a named color. Always returns full alpha.
func parseColor(s string) (color.NRGBA, error) {
	if len(s) > 0 && s[0] == '#' {
		return parseHex(s[1:])
	}
	if c, ok := named[strings.ToLower(s)]; ok {
		return c, nil
	}
	return color.NRGBA{}, fmt.Errorf("unknown color %q", s)
}

func parseHex(h string) (color.NRGBA, error) {
	var r, g, b uint8
	switch len(h) {
	case 3: // #rgb -> expand each nibble
		v, err := strconv.ParseUint(h, 16, 32)
		if err != nil {
			return color.NRGBA{}, fmt.Errorf("bad short hex %q", h)
		}
		r = uint8((v>>8)&0xF) * 0x11
		g = uint8((v>>4)&0xF) * 0x11
		b = uint8(v&0xF) * 0x11
	case 6:
		v, err := strconv.ParseUint(h, 16, 32)
		if err != nil {
			return color.NRGBA{}, fmt.Errorf("bad hex %q", h)
		}
		r = uint8((v >> 16) & 0xFF)
		g = uint8((v >> 8) & 0xFF)
		b = uint8(v & 0xFF)
	default:
		return color.NRGBA{}, fmt.Errorf("hex must be #rgb or #rrggbb, got %q", h)
	}
	return color.NRGBA{R: r, G: g, B: b, A: 0xFF}, nil
}
