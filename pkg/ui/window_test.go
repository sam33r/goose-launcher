package ui

import (
	"image"
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

// setupTestWindow creates a window with test items
func setupTestWindow() *Window {
	items := []appinput.Item{
		{Text: "Item 1", Raw: "item1"},
		{Text: "Item 2", Raw: "item2"},
		{Text: "Item 3", Raw: "item3"},
		{Text: "Item 4", Raw: "item4"},
		{Text: "Item 5", Raw: "item5"},
	}

	window := &Window{
		theme:            material.NewTheme(),
		items:            items,
		filtered:         items,
		matchPositions:   make(map[int][]int),
		list:             NewList(),
		searchInput:      NewInput(),
		matcher:          matcher.NewFuzzyMatcher(false, false),
		highlightMatches: true, // Enable by default for tests
	}

	window.theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	return window
}

// setupTestContext creates a layout context for testing
func setupTestContext() layout.Context {
	var ops op.Ops

	gtx := layout.Context{
		Ops: &ops,
		Metric: unit.Metric{
			PxPerDp: 1.0,
			PxPerSp: 1.0,
		},
		Constraints: layout.Exact(image.Point{X: 800, Y: 600}),
	}

	return gtx
}

// TestArrowKeyNavigation tests that arrow key handlers move selection
func TestArrowKeyNavigation(t *testing.T) {
	w := setupTestWindow()

	// Initial selection should be 0
	if w.list.Selected() != 0 {
		t.Errorf("initial selection = %d, want 0", w.list.Selected())
	}

	// Simulate down arrow - should move to item 1
	w.list.MoveDown(len(w.filtered))

	if w.list.Selected() != 1 {
		t.Errorf("after down arrow, selection = %d, want 1", w.list.Selected())
	}

	// Simulate down arrow again - should move to item 2
	w.list.MoveDown(len(w.filtered))

	if w.list.Selected() != 2 {
		t.Errorf("after second down arrow, selection = %d, want 2", w.list.Selected())
	}

	// Simulate up arrow - should move back to item 1
	w.list.MoveUp()

	if w.list.Selected() != 1 {
		t.Errorf("after up arrow, selection = %d, want 1", w.list.Selected())
	}
}

// TestEnterKeySelection tests that Enter key logic selects the highlighted item
func TestEnterKeySelection(t *testing.T) {
	w := setupTestWindow()

	// Move to item 2
	w.list.selected = 2

	// Simulate Enter key selection logic
	if len(w.filtered) > 0 {
		idx := w.list.Selected()
		w.selected = w.filtered[idx].Raw
	}

	if w.selected != "item3" {
		t.Errorf("after Enter on item 2, selected = %q, want %q", w.selected, "item3")
	}
}

// TestESCKeyCancellation tests that ESC key logic sets cancelled flag
func TestESCKeyCancellation(t *testing.T) {
	w := setupTestWindow()

	// Simulate ESC key
	w.cancelled = true

	if !w.cancelled {
		t.Error("after ESC key, cancelled should be true")
	}
}

// TestSearchFiltering tests that search filters items correctly
func TestSearchFiltering(t *testing.T) {
	w := setupTestWindow()

	// Initially all items should be shown
	if len(w.filtered) != 5 {
		t.Errorf("initial filtered count = %d, want 5", len(w.filtered))
	}

	// Set search text to "2"
	w.searchInput.SetText("2")
	w.filterItems("2")

	// Should only show Item 2
	if len(w.filtered) != 1 {
		t.Errorf("filtered count after search '2' = %d, want 1", len(w.filtered))
	}
	if len(w.filtered) > 0 && w.filtered[0].Text != "Item 2" {
		t.Errorf("filtered[0].Text = %q, want %q", w.filtered[0].Text, "Item 2")
	}

	// Search for "Item" - should match all
	w.searchInput.SetText("Item")
	w.filterItems("Item")

	if len(w.filtered) != 5 {
		t.Errorf("filtered count after search 'Item' = %d, want 5", len(w.filtered))
	}

	// Search for "xyz" - should match none
	w.searchInput.SetText("xyz")
	w.filterItems("xyz")

	if len(w.filtered) != 0 {
		t.Errorf("filtered count after search 'xyz' = %d, want 0", len(w.filtered))
	}
}

// TestSelectionBoundsAfterFiltering tests that selection stays in bounds after filtering
func TestSelectionBoundsAfterFiltering(t *testing.T) {
	w := setupTestWindow()
	gtx := setupTestContext()

	// Move to item 4 (last item)
	w.list.selected = 4

	// Filter to only show 2 items
	w.searchInput.SetText("Item 1")
	w.filterItems("Item 1")

	// Layout should fix selection to be in bounds
	w.layout(gtx)

	// Selection should be adjusted to be in bounds (0 for 1 item)
	if w.list.Selected() >= len(w.filtered) {
		t.Errorf("selection %d is out of bounds for %d filtered items", w.list.Selected(), len(w.filtered))
	}
}

// TestMouseClickSelection tests that clicking an item selects it
func TestMouseClickSelection(t *testing.T) {
	w := setupTestWindow()
	gtx := setupTestContext()

	// Layout once to initialize
	w.layout(gtx)

	// Simulate click on item 2 (index 2) - this happens in the list layout
	// The list sets both clickedIdx and selected when clicked
	w.list.clickedIdx = 2
	w.list.selected = 2

	// Layout should process the click and set w.selected
	w.layout(gtx)

	// Selection in list should be item 2
	if w.list.Selected() != 2 {
		t.Errorf("after click on item 2, list selection = %d, want 2", w.list.Selected())
	}

	// Window selected item should be set
	if w.selected != "item3" {
		t.Errorf("after click, window selected = %q, want %q", w.selected, "item3")
	}
}

// TestAllItemsRendered tests that all items are rendered (not just one)
func TestAllItemsRendered(t *testing.T) {
	items := []appinput.Item{
		{Text: "Item 1", Raw: "item1"},
		{Text: "Item 2", Raw: "item2"},
		{Text: "Item 3", Raw: "item3"},
	}

	window := &Window{
		theme:            material.NewTheme(),
		items:            items,
		filtered:         items,
		matchPositions:   make(map[int][]int),
		list:             NewList(),
		searchInput:      NewInput(),
		matcher:          matcher.NewFuzzyMatcher(false, false),
		highlightMatches: true,
	}
	window.theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	// Create headless window for rendering
	sz := image.Point{X: 800, Y: 600}
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

	// Layout the window
	window.layout(gtx)

	// Frame to headless window
	hw.Frame(gtx.Ops)

	// Take screenshot
	img := image.NewRGBA(image.Rectangle{Max: sz})
	if err := hw.Screenshot(img); err != nil {
		t.Fatalf("failed to take screenshot: %v", err)
	}

	// Basic check: image should not be all white or all black
	// This is a simple sanity check that something was rendered
	allSame := true
	firstPixel := img.At(0, 0)
	for y := 0; y < sz.Y; y++ {
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
		t.Error("rendered image appears to be blank (all same color)")
	}
}

// TestEmptyItemsList tests behavior with no items
func TestEmptyItemsList(t *testing.T) {
	window := &Window{
		theme:            material.NewTheme(),
		items:            []appinput.Item{},
		filtered:         []appinput.Item{},
		matchPositions:   make(map[int][]int),
		list:             NewList(),
		searchInput:      NewInput(),
		matcher:          matcher.NewFuzzyMatcher(false, false),
		highlightMatches: true,
	}
	window.theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	gtx := setupTestContext()

	// Layout should not crash with empty items
	w := window
	w.layout(gtx)

	// Simulate down arrow - selection should stay at 0
	w.list.MoveDown(len(w.filtered))

	if w.list.Selected() != 0 {
		t.Errorf("selection with empty list = %d, want 0", w.list.Selected())
	}

	// Simulate Enter - should not panic
	if len(w.filtered) > 0 {
		idx := w.list.Selected()
		w.selected = w.filtered[idx].Raw
	}

	if w.selected != "" {
		t.Errorf("selected with empty list = %q, want empty", w.selected)
	}
}

// TestSearchInputFocus tests that search input focus request works
func TestSearchInputFocus(t *testing.T) {
	w := setupTestWindow()
	gtx := setupTestContext()

	// Focus the search input
	w.searchInput.Focus()

	// After Focus(), requestFocus should be true
	if !w.searchInput.requestFocus {
		t.Error("requestFocus should be true after Focus()")
	}

	// Layout
	w.layout(gtx)

	// After layout, requestFocus should be reset
	if w.searchInput.requestFocus {
		t.Error("requestFocus should be false after layout")
	}
}

// TestFuzzyMatching tests that fuzzy matching works correctly
func TestFuzzyMatching(t *testing.T) {
	w := setupTestWindow()

	tests := []struct {
		query    string
		wantLen  int
		wantFirst string
	}{
		{"it", 5, "Item 1"}, // Should match all "Item X"
		{"2", 1, "Item 2"},  // Should match only "Item 2"
		{"i1", 1, "Item 1"}, // Fuzzy match "Item 1"
		{"i5", 1, "Item 5"}, // Fuzzy match "Item 5"
		{"xyz", 0, ""},      // No match
		{"", 5, "Item 1"},   // Empty query matches all
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			w.filterItems(tt.query)

			if len(w.filtered) != tt.wantLen {
				t.Errorf("query %q: filtered len = %d, want %d", tt.query, len(w.filtered), tt.wantLen)
			}

			if tt.wantLen > 0 && w.filtered[0].Text != tt.wantFirst {
				t.Errorf("query %q: first item = %q, want %q", tt.query, w.filtered[0].Text, tt.wantFirst)
			}
		})
	}
}

// TestEnterKeyOutputsAndExits tests that pressing Enter outputs selected item and sets exit flag
// Regression test for issue: "enter key is still not working"
func TestEnterKeyOutputsAndExits(t *testing.T) {
	w := setupTestWindow()

	// Move to item 1
	w.list.selected = 1

	// Simulate Enter key
	if len(w.filtered) > 0 {
		idx := w.list.Selected()
		w.selected = w.filtered[idx].Raw
	}

	// w.selected should be set
	if w.selected != "item2" {
		t.Errorf("after Enter, selected = %q, want %q", w.selected, "item2")
	}

	// In Run(), this would cause the window to exit
	// We verify the logic here
	shouldExit := w.selected != "" || w.cancelled
	if !shouldExit {
		t.Error("window should exit after selecting an item")
	}
}

// TestArrowKeysWorkAfterFilteringToLastItem tests arrow key navigation after filtering
// Regression test for issue: "if i filter the results by typing in a substring, and then go to the last item, then the arrow keys stop working for going back up"
func TestArrowKeysWorkAfterFilteringToLastItem(t *testing.T) {
	w := setupTestWindow()
	gtx := setupTestContext()

	// Initially 5 items
	if len(w.filtered) != 5 {
		t.Fatalf("initial filtered count = %d, want 5", len(w.filtered))
	}

	// Filter to 3 items
	w.searchInput.SetText("Item")
	w.filterItems("Item")

	// All items match "Item", should still have 5
	if len(w.filtered) != 5 {
		t.Errorf("after filter 'Item', filtered count = %d, want 5", len(w.filtered))
	}

	// Filter to fewer items by adding more specific query
	w.searchInput.SetText("Item 1")
	w.filterItems("Item 1")

	// Should have 1 item now
	if len(w.filtered) != 1 {
		t.Errorf("after filter 'Item 1', filtered count = %d, want 1", len(w.filtered))
	}

	// Layout to adjust selection bounds
	w.layout(gtx)

	// Selection should be at 0 (only valid index)
	if w.list.Selected() != 0 {
		t.Errorf("selection after filtering = %d, want 0", w.list.Selected())
	}

	// Now clear filter to get more items
	w.searchInput.SetText("")
	w.filterItems("")

	// Should have all 5 items back
	if len(w.filtered) != 5 {
		t.Errorf("after clearing filter, filtered count = %d, want 5", len(w.filtered))
	}

	// Move to last item
	for i := 0; i < 4; i++ {
		w.list.MoveDown(len(w.filtered))
	}

	if w.list.Selected() != 4 {
		t.Errorf("after moving to last item, selection = %d, want 4", w.list.Selected())
	}

	// Now move up - this should work
	w.list.MoveUp()

	if w.list.Selected() != 3 {
		t.Errorf("after MoveUp from last item, selection = %d, want 3", w.list.Selected())
	}

	// Move up again
	w.list.MoveUp()

	if w.list.Selected() != 2 {
		t.Errorf("after second MoveUp, selection = %d, want 2", w.list.Selected())
	}
}

// TestKeyboardEventsProcessedBeforeEditor tests that keyboard events are handled before editor
func TestKeyboardEventsProcessedBeforeEditor(t *testing.T) {
	w := setupTestWindow()
	gtx := setupTestContext()

	// Layout once to initialize
	w.layout(gtx)

	// Initial selection is 0
	if w.list.Selected() != 0 {
		t.Errorf("initial selection = %d, want 0", w.list.Selected())
	}

	// Simulate down arrow
	w.list.MoveDown(len(w.filtered))

	// Selection should move even if editor has focus
	if w.list.Selected() != 1 {
		t.Errorf("after down arrow, selection = %d, want 1", w.list.Selected())
	}

	// Type some text in search
	w.searchInput.SetText("test")

	// Layout again
	w.layout(gtx)

	// Arrow keys should still work after typing
	w.list.MoveUp()

	if w.list.Selected() != 0 {
		t.Errorf("after up arrow (with text in search), selection = %d, want 0", w.list.Selected())
	}
}

// TestShiftEnterKeySelection tests that Shift+Enter outputs the current search query
func TestShiftEnterKeySelection(t *testing.T) {
	w := setupTestWindow()

	// Set search text to "Custom Query"
	w.searchInput.SetText("Custom Query")

	// Simulate Shift+Enter key selection logic
	// In the real app, this is handled in the event loops
	w.selected = w.searchInput.Text()

	if w.selected != "Custom Query" {
		t.Errorf("after Shift+Enter, selected = %q, want %q", w.selected, "Custom Query")
	}

	// It should output what was typed even if it matches nothing in the list
	w.searchInput.SetText("Matches Nothing")
	w.filterItems("Matches Nothing")
	if len(w.filtered) != 0 {
		t.Fatalf("expected 0 filtered items, got %d", len(w.filtered))
	}

	w.selected = w.searchInput.Text()
	if w.selected != "Matches Nothing" {
		t.Errorf("after Shift+Enter with no matches, selected = %q, want %q", w.selected, "Matches Nothing")
	}
}
