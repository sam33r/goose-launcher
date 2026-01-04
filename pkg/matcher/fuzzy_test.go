package matcher

import (
	"testing"

	"github.com/sam33r/goose-launcher/pkg/input"
)

func TestFuzzyMatch_SimpleMatch(t *testing.T) {
	matcher := NewFuzzyMatcher(false, false) // not case-sensitive, not exact

	item := input.Item{Text: "Downloads/file.txt", Index: 0}
	query := "dwn"

	match, positions := matcher.Match(query, item)

	if !match {
		t.Error("expected 'dwn' to match 'Downloads/file.txt'")
	}

	if len(positions) != 3 {
		t.Errorf("expected 3 match positions, got %d", len(positions))
	}

	// Should match: D-w-n
	expectedPositions := []int{0, 2, 3}
	for i, pos := range expectedPositions {
		if positions[i] != pos {
			t.Errorf("position %d: expected %d, got %d", i, pos, positions[i])
		}
	}
}

func TestFuzzyMatch_NoMatch(t *testing.T) {
	matcher := NewFuzzyMatcher(false, false)

	item := input.Item{Text: "Downloads/file.txt", Index: 0}
	query := "xyz"

	match, _ := matcher.Match(query, item)

	if match {
		t.Error("expected 'xyz' not to match 'Downloads/file.txt'")
	}
}

func TestFuzzyMatch_EmptyQuery(t *testing.T) {
	matcher := NewFuzzyMatcher(false, false)

	item := input.Item{Text: "anything", Index: 0}
	query := ""

	match, positions := matcher.Match(query, item)

	if !match {
		t.Error("expected empty query to match any item")
	}

	if len(positions) != 0 {
		t.Errorf("expected 0 positions for empty query, got %d", len(positions))
	}
}

func TestExactMatch(t *testing.T) {
	matcher := NewFuzzyMatcher(false, true) // exact mode

	item := input.Item{Text: "Documents/notes.txt", Index: 0}
	query := "notes"

	match, positions := matcher.Match(query, item)

	if !match {
		t.Error("expected 'notes' to match 'Documents/notes.txt' in exact mode")
	}

	if len(positions) != 5 {
		t.Errorf("expected 5 positions, got %d", len(positions))
	}
}

func TestCaseSensitiveMatch(t *testing.T) {
	matcher := NewFuzzyMatcher(true, false) // case-sensitive

	item := input.Item{Text: "Downloads", Index: 0}

	// Should match with correct case
	match1, _ := matcher.Match("Down", item)
	if !match1 {
		t.Error("expected 'Down' to match 'Downloads' (case-sensitive)")
	}

	// Should NOT match with wrong case
	match2, _ := matcher.Match("down", item)
	if match2 {
		t.Error("expected 'down' NOT to match 'Downloads' (case-sensitive)")
	}
}
