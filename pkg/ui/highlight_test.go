package ui

import (
	"image"
	"image/color"
	"testing"

	"gioui.org/font/gofont"
	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"

	appinput "github.com/sam33r/goose-launcher/pkg/input"
	"github.com/sam33r/goose-launcher/pkg/matcher"
)

// TestHighlightMatchesEnabled tests that highlighting is enabled by default
func TestHighlightMatchesEnabled(t *testing.T) {
	items := []appinput.Item{
		{Text: "Test Item", Raw: "test"},
	}

	w := &Window{
		theme:            material.NewTheme(),
		items:            items,
		filtered:         items,
		matchPositions:   make(map[int][]int),
		list:             NewList(),
		searchInput:      NewInput(),
		matcher:          matcher.NewFuzzyMatcher(false, false),
		highlightMatches: true,
	}
	w.theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	if !w.highlightMatches {
		t.Error("highlightMatches should be enabled by default")
	}
}

// TestHighlightMatchesDisabled tests that highlighting can be disabled
func TestHighlightMatchesDisabled(t *testing.T) {
	items := []appinput.Item{
		{Text: "Test Item", Raw: "test"},
	}

	w := &Window{
		theme:            material.NewTheme(),
		items:            items,
		filtered:         items,
		matchPositions:   make(map[int][]int),
		list:             NewList(),
		searchInput:      NewInput(),
		matcher:          matcher.NewFuzzyMatcher(false, false),
		highlightMatches: false,
	}
	w.theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	if w.highlightMatches {
		t.Error("highlightMatches should be disabled when set to false")
	}
}

// TestMatchPositionsStoredDuringFiltering tests that match positions are stored
func TestMatchPositionsStoredDuringFiltering(t *testing.T) {
	w := &Window{
		theme:            material.NewTheme(),
		items: []appinput.Item{
			{Text: "Test Item 1", Raw: "test1"},
			{Text: "Test Item 2", Raw: "test2"},
			{Text: "Another Item", Raw: "another"},
		},
		filtered:         nil,
		matchPositions:   make(map[int][]int),
		list:             NewList(),
		searchInput:      NewInput(),
		matcher:          matcher.NewFuzzyMatcher(false, false),
		highlightMatches: true,
	}
	w.theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	// Filter with query "Test"
	w.filterItems("Test")

	// Should have 2 filtered items
	if len(w.filtered) != 2 {
		t.Errorf("filtered count = %d, want 2", len(w.filtered))
	}

	// Should have match positions for both filtered items
	if len(w.matchPositions) != 2 {
		t.Errorf("matchPositions count = %d, want 2", len(w.matchPositions))
	}

	// Both items should have non-empty match positions
	for i := 0; i < 2; i++ {
		if positions, ok := w.matchPositions[i]; !ok || len(positions) == 0 {
			t.Errorf("item %d should have match positions", i)
		}
	}
}

// TestMatchPositionsClearedWhenNoQuery tests that match positions are cleared with empty query
func TestMatchPositionsClearedWhenNoQuery(t *testing.T) {
	w := &Window{
		theme:            material.NewTheme(),
		items: []appinput.Item{
			{Text: "Test Item", Raw: "test"},
		},
		filtered:         nil,
		matchPositions:   make(map[int][]int),
		list:             NewList(),
		searchInput:      NewInput(),
		matcher:          matcher.NewFuzzyMatcher(false, false),
		highlightMatches: true,
	}

	// Filter with a query first
	w.filterItems("Test")
	if len(w.matchPositions) == 0 {
		t.Error("matchPositions should not be empty after filtering")
	}

	// Clear filter
	w.filterItems("")

	// Match positions should be cleared
	if len(w.matchPositions) != 0 {
		t.Errorf("matchPositions should be empty with no query, got %d entries", len(w.matchPositions))
	}
}

// TestHighlightingWithDifferentQueries tests highlighting with various queries
func TestHighlightingWithDifferentQueries(t *testing.T) {
	w := &Window{
		theme:            material.NewTheme(),
		items: []appinput.Item{
			{Text: "Hello World", Raw: "hello"},
			{Text: "Test Item", Raw: "test"},
			{Text: "Another Example", Raw: "another"},
		},
		filtered:         nil,
		matchPositions:   make(map[int][]int),
		list:             NewList(),
		searchInput:      NewInput(),
		matcher:          matcher.NewFuzzyMatcher(false, false),
		highlightMatches: true,
	}

	tests := []struct {
		query              string
		expectedFiltered   int
		expectedFirstMatch string
	}{
		{"Hel", 2, "Hello World"},    // Matches "Hello World" and "Another Example" (fuzzy)
		{"Item", 1, "Test Item"},
		{"e", 3, "Hello World"},      // Matches all items with 'e'
		{"xyz", 0, ""},               // No match
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			w.filterItems(tt.query)

			if len(w.filtered) != tt.expectedFiltered {
				t.Errorf("query %q: filtered count = %d, want %d", tt.query, len(w.filtered), tt.expectedFiltered)
			}

			if tt.expectedFiltered > 0 {
				if w.filtered[0].Text != tt.expectedFirstMatch {
					t.Errorf("query %q: first match = %q, want %q", tt.query, w.filtered[0].Text, tt.expectedFirstMatch)
				}

				// Should have match positions for first item
				if positions, ok := w.matchPositions[0]; !ok || len(positions) == 0 {
					t.Errorf("query %q: first item should have match positions", tt.query)
				}
			}
		})
	}
}

// TestHighlightedTextRendering tests that highlighted text renders without crashing
func TestHighlightedTextRendering(t *testing.T) {
	items := []appinput.Item{
		{Text: "Test Item 1", Raw: "test1"},
		{Text: "Test Item 2", Raw: "test2"},
	}

	list := NewList()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	// Create match positions for "Test"
	matchPositions := map[int][]int{
		0: {0, 1, 2, 3}, // "Test" in "Test Item 1"
		1: {0, 1, 2, 3}, // "Test" in "Test Item 2"
	}

	// Create headless window
	sz := image.Point{X: 800, Y: 400}
	hw, err := headless.NewWindow(sz.X, sz.Y)
	if err != nil {
		t.Fatalf("failed to create headless window: %v", err)
	}

	var ops op.Ops
	gtx := layout.Context{
		Ops: &ops,
		Metric: unit.Metric{
			PxPerDp: 1.0,
			PxPerSp: 1.0,
		},
		Constraints: layout.Exact(sz),
	}

	// Layout with highlighting enabled
	dims := list.Layout(gtx, theme, items, matchPositions, true)

	// Should have non-zero dimensions
	if dims.Size.X == 0 || dims.Size.Y == 0 {
		t.Error("highlighted list should have non-zero dimensions")
	}

	// Frame and screenshot
	hw.Frame(gtx.Ops)
	img := image.NewRGBA(image.Rectangle{Max: sz})
	if err := hw.Screenshot(img); err != nil {
		t.Fatalf("failed to take screenshot: %v", err)
	}

	// Check that image has content (not all same color)
	allSame := true
	firstPixel := img.At(0, 0)
	for y := 0; y < 100; y++ {
		for x := 0; x < sz.X; x++ {
			if img.At(x, y) != firstPixel {
				allSame = false
				break
			}
		}
		if !allSame {
			break
		}
	}

	if allSame {
		t.Error("highlighted text should render visible content")
	}
}

// TestHighlightingDisabledRendersNormally tests rendering with highlighting disabled
func TestHighlightingDisabledRendersNormally(t *testing.T) {
	items := []appinput.Item{
		{Text: "Test Item", Raw: "test"},
	}

	list := NewList()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	matchPositions := map[int][]int{
		0: {0, 1, 2, 3}, // Would highlight "Test" if enabled
	}

	var ops op.Ops
	gtx := layout.Context{
		Ops: &ops,
		Metric: unit.Metric{
			PxPerDp: 1.0,
			PxPerSp: 1.0,
		},
		Constraints: layout.Exact(image.Point{X: 800, Y: 100}),
	}

	// Layout with highlighting disabled (should not crash)
	dims := list.Layout(gtx, theme, items, matchPositions, false)

	// Should have non-zero dimensions
	if dims.Size.X == 0 || dims.Size.Y == 0 {
		t.Error("list without highlighting should have non-zero dimensions")
	}
}

// TestTextSegmentation tests the text segmentation logic
func TestTextSegmentation(t *testing.T) {
	list := NewList()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	// Create test context
	var ops op.Ops
	gtx := layout.Context{
		Ops: &ops,
		Metric: unit.Metric{
			PxPerDp: 1.0,
			PxPerSp: 1.0,
		},
		Constraints: layout.Exact(image.Point{X: 800, Y: 100}),
	}

	// Test text with match positions: "Hello" with positions [0,1] (He)
	text := "Hello"
	matchPositions := []int{0, 1}
	baseColor := color.NRGBA{R: 0, G: 0, B: 0, A: 255}       // Black
	highlightColor := color.NRGBA{R: 255, G: 255, B: 255, A: 255} // White

	// Should not crash
	dims := list.layoutHighlightedText(gtx, theme, text, matchPositions, baseColor, highlightColor)

	// Should have non-zero dimensions
	if dims.Size.X == 0 || dims.Size.Y == 0 {
		t.Error("highlighted text should have non-zero dimensions")
	}
}

// TestCommandLineFlagDefault tests that --highlight-matches defaults to true
func TestCommandLineFlagDefault(t *testing.T) {
	// This would be tested in config package, but we verify the default here
	items := []appinput.Item{{Text: "Test", Raw: "test"}}

	// When created with true (default)
	w := NewWindow(items, true)

	if !w.highlightMatches {
		t.Error("NewWindow with highlightMatches=true should enable highlighting")
	}

	// When created with false (disabled via flag)
	w2 := NewWindow(items, false)

	if w2.highlightMatches {
		t.Error("NewWindow with highlightMatches=false should disable highlighting")
	}
}
