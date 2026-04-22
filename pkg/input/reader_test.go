package input

import (
	"strings"
	"testing"
)

func TestParseLine_WithSeparator(t *testing.T) {
	r := &Reader{}
	line := "files   . /home/user/document.txt"

	item := r.parseLine(line, 0)

	if item.Plugin != "files" {
		t.Errorf("expected plugin 'files', got '%s'", item.Plugin)
	}
	if item.Text != "/home/user/document.txt" {
		t.Errorf("expected text '/home/user/document.txt', got '%s'", item.Text)
	}
	if item.Raw != line {
		t.Errorf("expected raw '%s', got '%s'", line, item.Raw)
	}
	if item.Index != 0 {
		t.Errorf("expected index 0, got %d", item.Index)
	}
}

func TestParseLine_WithoutSeparator(t *testing.T) {
	r := &Reader{}
	line := "plain text item"

	item := r.parseLine(line, 5)

	if item.Plugin != "" {
		t.Errorf("expected empty plugin, got '%s'", item.Plugin)
	}
	if item.Text != "plain text item" {
		t.Errorf("expected text 'plain text item', got '%s'", item.Text)
	}
	if item.Index != 5 {
		t.Errorf("expected index 5, got %d", item.Index)
	}
}

func TestReadAll(t *testing.T) {
	input := `files   . /home/user/file1.txt
files   . /home/user/file2.txt
plain item without plugin
chrome   . https://example.com`

	reader := NewReader(strings.NewReader(input), "")
	items, err := reader.ReadAll()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(items))
	}

	// Test first item
	if items[0].Plugin != "files" {
		t.Errorf("item 0: expected plugin 'files', got '%s'", items[0].Plugin)
	}
	if items[0].Text != "/home/user/file1.txt" {
		t.Errorf("item 0: expected text '/home/user/file1.txt', got '%s'", items[0].Text)
	}
	if items[0].Index != 0 {
		t.Errorf("item 0: expected index 0, got %d", items[0].Index)
	}

	// Test item without plugin
	if items[2].Plugin != "" {
		t.Errorf("item 2: expected empty plugin, got '%s'", items[2].Plugin)
	}
	if items[2].Text != "plain item without plugin" {
		t.Errorf("item 2: expected text 'plain item without plugin', got '%s'", items[2].Text)
	}
}

func TestReadAll_PangoMarkup(t *testing.T) {
	input := "files   . <b>bold</b> path\n" +
		"<i>italic</i> plain\n" +
		"<unterminated\n"

	reader := NewReader(strings.NewReader(input), "pango")
	items, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// Markup stripped from Text; Spans populated; Raw stays as the original input line.
	if items[0].Text != "bold path" {
		t.Errorf("item[0].Text = %q, want %q", items[0].Text, "bold path")
	}
	if items[0].Raw != "files   . <b>bold</b> path" {
		t.Errorf("item[0].Raw = %q, want original input line %q", items[0].Raw, "files   . <b>bold</b> path")
	}
	if len(items[0].Spans) == 0 || !items[0].Spans[0].Bold {
		t.Errorf("item[0].Spans = %+v, want leading bold span", items[0].Spans)
	}

	// No plugin prefix: Raw is the original input line.
	if items[1].Text != "italic plain" {
		t.Errorf("item[1].Text = %q", items[1].Text)
	}
	if items[1].Raw != "<i>italic</i> plain" {
		t.Errorf("item[1].Raw = %q, want %q", items[1].Raw, "<i>italic</i> plain")
	}

	// Malformed falls back to literal text with no Spans — one bad line
	// must not break the whole launcher.
	if items[2].Spans != nil {
		t.Errorf("item[2].Spans = %+v, want nil on parse fallback", items[2].Spans)
	}
	if items[2].Text != "<unterminated" {
		t.Errorf("item[2].Text = %q, want literal fallback", items[2].Text)
	}
}

func TestReadAll_EmptyInput(t *testing.T) {
	reader := NewReader(strings.NewReader(""), "")
	items, err := reader.ReadAll()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}
