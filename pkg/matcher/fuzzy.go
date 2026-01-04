package matcher

import (
	"strings"

	"github.com/sam33r/goose-launcher/pkg/input"
)

// FuzzyMatcher performs fuzzy string matching
type FuzzyMatcher struct {
	caseSensitive bool
	exact         bool
}

// NewFuzzyMatcher creates a new fuzzy matcher
func NewFuzzyMatcher(caseSensitive, exact bool) *FuzzyMatcher {
	return &FuzzyMatcher{
		caseSensitive: caseSensitive,
		exact:         exact,
	}
}

// Match checks if query matches the item's text
// Returns: (matched bool, positions []int)
func (m *FuzzyMatcher) Match(query string, item input.Item) (bool, []int) {
	if query == "" {
		// Empty query matches everything
		return true, []int{}
	}

	if m.exact {
		return m.exactMatch(query, item.Text)
	}

	return m.fuzzyMatch(query, item.Text)
}

// fuzzyMatch performs fuzzy matching (simplified fzf algorithm)
func (m *FuzzyMatcher) fuzzyMatch(query, text string) (bool, []int) {
	queryRunes := []rune(query)
	textRunes := []rune(text)

	if !m.caseSensitive {
		queryRunes = []rune(strings.ToLower(query))
		textRunes = []rune(strings.ToLower(text))
	}

	positions := make([]int, 0, len(queryRunes))
	textIdx := 0

	// Try to find each query character in order
	for _, qChar := range queryRunes {
		found := false

		for textIdx < len(textRunes) {
			if textRunes[textIdx] == qChar {
				positions = append(positions, textIdx)
				textIdx++
				found = true
				break
			}
			textIdx++
		}

		if !found {
			// Query character not found in text
			return false, nil
		}
	}

	return true, positions
}

// exactMatch performs exact substring matching
func (m *FuzzyMatcher) exactMatch(query, text string) (bool, []int) {
	searchText := text
	searchQuery := query

	if !m.caseSensitive {
		searchText = strings.ToLower(text)
		searchQuery = strings.ToLower(query)
	}

	idx := strings.Index(searchText, searchQuery)
	if idx == -1 {
		return false, nil
	}

	// Create positions array for all characters in match
	positions := make([]int, len(query))
	for i := range positions {
		positions[i] = idx + i
	}

	return true, positions
}
