package input

import (
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
