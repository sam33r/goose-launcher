package input

// Item represents a single selectable item from stdin
type Item struct {
	Plugin string // Plugin name before separator (e.g., "files")
	Text   string // Item text after separator
	Raw    string // Full line from stdin
	Index  int    // Original order from stdin
}
