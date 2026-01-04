package input

import (
	"bufio"
	"io"
	"strings"
)

const separator = "   . " // 3 spaces + dot + space

// Reader reads and parses items from stdin
type Reader struct {
	scanner *bufio.Scanner
}

// NewReader creates a new Reader from an io.Reader
func NewReader(r io.Reader) *Reader {
	return &Reader{
		scanner: bufio.NewScanner(r),
	}
}

// parseLine parses a single line into an Item
// Format: "plugin   . item_text" or just "item_text"
func (r *Reader) parseLine(line string, index int) Item {
	parts := strings.SplitN(line, separator, 2)

	if len(parts) == 2 {
		// Has separator: "plugin   . text"
		return Item{
			Plugin: strings.TrimSpace(parts[0]),
			Text:   parts[1],
			Raw:    line,
			Index:  index,
		}
	}

	// No separator: entire line is text
	return Item{
		Plugin: "",
		Text:   line,
		Raw:    line,
		Index:  index,
	}
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
