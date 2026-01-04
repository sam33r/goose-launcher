package input

import "testing"

func TestItemCreation(t *testing.T) {
	item := Item{
		Plugin: "files",
		Text:   "/path/to/file.txt",
		Raw:    "files   . /path/to/file.txt",
		Index:  0,
	}

	if item.Plugin != "files" {
		t.Errorf("expected plugin 'files', got '%s'", item.Plugin)
	}
	if item.Text != "/path/to/file.txt" {
		t.Errorf("expected text '/path/to/file.txt', got '%s'", item.Text)
	}
}
