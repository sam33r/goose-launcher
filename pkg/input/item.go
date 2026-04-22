package input

import "github.com/sam33r/goose-launcher/pkg/markup"

// Item represents a single selectable item from stdin
type Item struct {
	Plugin string // Plugin name before separator (e.g., "files")
	Text   string // Item text after separator (markup stripped when Spans is non-nil)
	Raw    string // Original input line, returned verbatim to the caller on selection (markup intact when Spans is non-nil)
	Index  int    // Original order from stdin
	Spans  []markup.Span // Styled runs covering Text; nil when markup is disabled or parse fell back.
}
