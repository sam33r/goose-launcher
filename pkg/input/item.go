package input

import (
	"strings"

	"github.com/sam33r/goose-launcher/pkg/markup"
)

// Item represents a single selectable item from stdin
type Item struct {
	Plugin string // Plugin name before separator (e.g., "files")
	Text   string // Item text after separator (markup stripped when Spans is non-nil)
	Raw    string // Original input line, returned verbatim to the caller on selection (markup intact when Spans is non-nil)
	Index  int    // Original order from stdin
	Spans  []markup.Span // Styled runs covering Text; nil when markup is disabled or parse fell back.

	// LowerText is Text lowercased once at parse time so per-keystroke matching
	// doesn't pay the strings.ToLower allocation per item per call.
	// Equal to Text when Text is already all-ASCII-lowercase.
	LowerText string
	// ASCII is true when Text is pure ASCII; lets the matcher take a byte-level
	// fast path that avoids []rune conversion (the dominant cost on large inputs).
	ASCII bool
}

// Init populates LowerText and ASCII from Text. Reader calls this; tests that
// build Items by hand can call it (or leave it — matcher falls back gracefully).
func (i *Item) Init() {
	i.ASCII = isASCII(i.Text)
	if i.ASCII {
		i.LowerText = asciiToLower(i.Text)
	} else {
		i.LowerText = strings.ToLower(i.Text)
	}
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// asciiToLower is allocation-free when s is already lowercase, otherwise
// allocates exactly one new string. Faster than strings.ToLower for ASCII.
func asciiToLower(s string) string {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			b := []byte(s)
			b[i] = c + 32
			for j := i + 1; j < len(b); j++ {
				if b[j] >= 'A' && b[j] <= 'Z' {
					b[j] += 32
				}
			}
			return string(b)
		}
	}
	return s
}
