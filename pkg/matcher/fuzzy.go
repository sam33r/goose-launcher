package matcher

import (
	"strings"

	"github.com/sam33r/goose-launcher/pkg/input"
)

// FuzzyMatcher performs fuzzy / exact substring matching.
//
// Hot path notes (matching runs N items × every keystroke):
//   - We rely on input.Item.LowerText / ASCII, populated once at parse time,
//     to avoid per-call strings.ToLower allocations.
//   - When both query and text are ASCII we operate on bytes directly and
//     skip []rune conversion entirely (the dominant alloc on large inputs).
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

// Match checks if query matches the item's text and returns match positions
// (rune indices into item.Text) for highlighting.
func (m *FuzzyMatcher) Match(query string, item input.Item) (bool, []int) {
	return m.match(query, item, true)
}

// MatchOnly is a position-free fast path for callers (e.g. counting) that
// don't need highlight/rank positions. Saves the positions allocation.
func (m *FuzzyMatcher) MatchOnly(query string, item input.Item) bool {
	ok, _ := m.match(query, item, false)
	return ok
}

func (m *FuzzyMatcher) match(query string, item input.Item, withPositions bool) (bool, []int) {
	if query == "" {
		return true, nil
	}

	text, lowerText := item.Text, item.LowerText
	ascii := item.ASCII
	if lowerText == "" && !item.ASCII {
		// Item built without Init() (e.g. legacy callers / hand-rolled tests).
		// Fall back to the cold path so behavior stays correct.
		ascii = isASCII(text) && isASCII(query)
		if !m.caseSensitive {
			lowerText = strings.ToLower(text)
		} else {
			lowerText = text
		}
	}

	searchText := text
	searchQuery := query
	if !m.caseSensitive {
		searchText = lowerText
		searchQuery = strings.ToLower(query)
	}

	if m.exact {
		return exactMatch(searchText, searchQuery, withPositions)
	}

	// ASCII byte-level fast path: positions are byte == rune indices.
	if ascii && isASCII(searchQuery) {
		return fuzzyMatchASCII(searchText, searchQuery, withPositions)
	}
	return fuzzyMatchRunes(searchText, searchQuery, withPositions)
}

// fuzzyMatchASCII walks the bytes of searchText looking for each byte of
// searchQuery in order. Both inputs must be pure ASCII.
func fuzzyMatchASCII(searchText, searchQuery string, withPositions bool) (bool, []int) {
	var positions []int
	if withPositions {
		positions = make([]int, 0, len(searchQuery))
	}
	textIdx := 0
	for qi := 0; qi < len(searchQuery); qi++ {
		qc := searchQuery[qi]
		found := false
		for textIdx < len(searchText) {
			if searchText[textIdx] == qc {
				if withPositions {
					positions = append(positions, textIdx)
				}
				textIdx++
				found = true
				break
			}
			textIdx++
		}
		if !found {
			return false, nil
		}
	}
	return true, positions
}

// fuzzyMatchRunes is the cold path for inputs containing non-ASCII runes.
func fuzzyMatchRunes(searchText, searchQuery string, withPositions bool) (bool, []int) {
	queryRunes := []rune(searchQuery)
	textRunes := []rune(searchText)

	var positions []int
	if withPositions {
		positions = make([]int, 0, len(queryRunes))
	}
	textIdx := 0
	for _, qChar := range queryRunes {
		found := false
		for textIdx < len(textRunes) {
			if textRunes[textIdx] == qChar {
				if withPositions {
					positions = append(positions, textIdx)
				}
				textIdx++
				found = true
				break
			}
			textIdx++
		}
		if !found {
			return false, nil
		}
	}
	return true, positions
}

// exactMatch is the substring-match path. Positions are rune indices into the
// pre-lowercased text; for ASCII these equal byte indices, which is the only
// case in practice. For non-ASCII text we map byte->rune index.
func exactMatch(searchText, searchQuery string, withPositions bool) (bool, []int) {
	idx := strings.Index(searchText, searchQuery)
	if idx == -1 {
		return false, nil
	}
	if !withPositions {
		return true, nil
	}

	// Fast path: pure ASCII (overwhelming common case) — byte == rune index.
	if isASCII(searchText) && isASCII(searchQuery) {
		positions := make([]int, len(searchQuery))
		for i := range positions {
			positions[i] = idx + i
		}
		return true, positions
	}

	// Cold path: convert byte idx to rune idx and span len(query) runes.
	queryRuneLen := 0
	for range searchQuery {
		queryRuneLen++
	}
	startRune := 0
	for bi := range searchText {
		if bi == idx {
			break
		}
		startRune++
	}
	positions := make([]int, queryRuneLen)
	for i := range positions {
		positions[i] = startRune + i
	}
	return true, positions
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}
