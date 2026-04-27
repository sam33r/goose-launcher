package input

import (
	"bufio"
	"io"
	"strings"

	"github.com/sam33r/goose-launcher/pkg/markup"
)

const separator = "   . " // 3 spaces + dot + space

// Reader reads and parses items from stdin
type Reader struct {
	scanner *bufio.Scanner
	markup  string // "" (off) or "pango"
}

// NewReader creates a new Reader from an io.Reader. The markup argument
// selects stdin markup parsing; pass "" to disable.
func NewReader(r io.Reader, markupFormat string) *Reader {
	return &Reader{
		scanner: bufio.NewScanner(r),
		markup:  markupFormat,
	}
}

// ParseLine parses a single line into an Item.
// Format: "plugin   . item_text" or just "item_text".
// markupFormat selects stdin markup parsing; pass "" to disable.
//
// Used directly by the daemon's streaming chunk handler so it can parse lines
// as they arrive without holding a Reader. Reader.ReadAll delegates here too.
func ParseLine(line string, index int, markupFormat string) Item {
	parts := strings.SplitN(line, separator, 2)

	var plugin, text string
	if len(parts) == 2 {
		plugin = strings.TrimSpace(parts[0])
		text = parts[1]
	} else {
		text = line
	}

	item := Item{
		Plugin: plugin,
		Text:   text,
		Raw:    line,
		Index:  index,
	}

	if markupFormat == "pango" {
		// Parse the text portion for display. On failure fall back to the
		// literal line — one bad item shouldn't break the whole launcher.
		// item.Raw stays as the original input line so the caller gets the
		// markup-bearing line verbatim — required for exact-line matching
		// in downstream history filters.
		plain, spans, err := markup.Parse(text)
		if err == nil {
			item.Text = plain
			item.Spans = spans
		}
	}

	item.Init()
	return item
}

// parseLine is kept as a thin method-receiver shim so existing tests
// (TestParseLine_*) continue to call r.parseLine(line, index).
func (r *Reader) parseLine(line string, index int) Item {
	return ParseLine(line, index, r.markup)
}

// ReadAll reads all items from stdin (blocking)
func (r *Reader) ReadAll() ([]Item, error) {
	var items []Item
	index := 0

	for r.scanner.Scan() {
		line := r.scanner.Text()
		items = append(items, r.parseLine(line, index))
		index++
	}

	if err := r.scanner.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
