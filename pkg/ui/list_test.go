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
)

// TestListMoveUp tests moving selection up
func TestListMoveUp(t *testing.T) {
	list := NewList()

	// Start at item 2
	list.selected = 2

	// Move up
	list.MoveUp()

	if list.selected != 1 {
		t.Errorf("after MoveUp from 2, selected = %d, want 1", list.selected)
	}

	// Move up again
	list.MoveUp()

	if list.selected != 0 {
		t.Errorf("after MoveUp from 1, selected = %d, want 0", list.selected)
	}

	// Try to move up from 0 - should stay at 0
	list.MoveUp()

	if list.selected != 0 {
		t.Errorf("after MoveUp from 0, selected = %d, want 0", list.selected)
	}
}

// TestListMoveDown tests moving selection down
func TestListMoveDown(t *testing.T) {
	list := NewList()
	itemCount := 5

	// Start at item 0
	list.selected = 0

	// Move down
	list.MoveDown(itemCount)

	if list.selected != 1 {
		t.Errorf("after MoveDown from 0, selected = %d, want 1", list.selected)
	}

	// Move down multiple times
	list.MoveDown(itemCount)
	list.MoveDown(itemCount)
	list.MoveDown(itemCount)

	if list.selected != 4 {
		t.Errorf("after multiple MoveDowns, selected = %d, want 4", list.selected)
	}

	// Try to move down from last item - should stay at 4
	list.MoveDown(itemCount)

	if list.selected != 4 {
		t.Errorf("after MoveDown from last item, selected = %d, want 4", list.selected)
	}
}

// TestListClickDetection tests click handling
func TestListClickDetection(t *testing.T) {
	list := NewList()

	// Initially no click
	if list.GetClicked() != -1 {
		t.Errorf("initial clicked = %d, want -1", list.GetClicked())
	}

	// Simulate click on item 2
	list.clickedIdx = 2

	if list.GetClicked() != 2 {
		t.Errorf("after click on item 2, clicked = %d, want 2", list.GetClicked())
	}

	// Reset click
	list.ResetClicked()

	if list.GetClicked() != -1 {
		t.Errorf("after ResetClicked, clicked = %d, want -1", list.GetClicked())
	}
}

// TestListSelectionBounds tests that selection stays within bounds
func TestListSelectionBounds(t *testing.T) {
	items := []appinput.Item{
		{Text: "Item 1", Raw: "item1"},
		{Text: "Item 2", Raw: "item2"},
		{Text: "Item 3", Raw: "item3"},
	}

	list := NewList()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	var ops op.Ops
	gtx := layout.Context{
		Ops: &ops,
		Metric: unit.Metric{
			PxPerDp: 1.0,
			PxPerSp: 1.0,
		},
		Constraints: layout.Exact(image.Point{X: 800, Y: 600}),
	}

	// Set selection out of bounds (high)
	list.selected = 10

	// Layout should fix it
	list.Layout(gtx, theme, items, make(map[int][]int), false)

	if list.selected != 2 {
		t.Errorf("after layout with selected=10 and 3 items, selected = %d, want 2", list.selected)
	}

	// Set selection out of bounds (negative)
	list.selected = -5

	// Layout should fix it
	list.Layout(gtx, theme, items, make(map[int][]int), false)

	if list.selected != 0 {
		t.Errorf("after layout with selected=-5, selected = %d, want 0", list.selected)
	}
}

// TestListLayoutEmptyItems tests list layout with no items
func TestListLayoutEmptyItems(t *testing.T) {
	items := []appinput.Item{}

	list := NewList()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	var ops op.Ops
	gtx := layout.Context{
		Ops: &ops,
		Metric: unit.Metric{
			PxPerDp: 1.0,
			PxPerSp: 1.0,
		},
		Constraints: layout.Exact(image.Point{X: 800, Y: 600}),
	}

	// Should not panic with empty items
	dims := list.Layout(gtx, theme, items, make(map[int][]int), false)

	// Dimensions should be zero for empty list
	if dims.Size.X != 0 || dims.Size.Y != 0 {
		t.Errorf("empty list dimensions = %v, want zero", dims.Size)
	}
}

// TestListRendersAllItems tests that all items are rendered (issue: only one item showing)
func TestListRendersAllItems(t *testing.T) {
	items := []appinput.Item{
		{Text: "Item 1", Raw: "item1"},
		{Text: "Item 2", Raw: "item2"},
		{Text: "Item 3", Raw: "item3"},
		{Text: "Item 4", Raw: "item4"},
		{Text: "Item 5", Raw: "item5"},
	}

	list := NewList()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	// Create headless window
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

	// Layout all items
	list.Layout(gtx, theme, items, make(map[int][]int), false)

	// Frame and screenshot
	hw.Frame(gtx.Ops)
	img := image.NewRGBA(image.Rectangle{Max: sz})
	if err := hw.Screenshot(img); err != nil {
		t.Fatalf("failed to take screenshot: %v", err)
	}

	// Check that image has content (not all same color)
	// This ensures items were rendered
	allSame := true
	firstPixel := img.At(0, 0)
	for y := 0; y < 100; y++ { // Check first 100 rows
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
		t.Error("list appears blank - items may not be rendering")
	}
}

// TestListSelectionHighlight tests that selected item has background color
func TestListSelectionHighlight(t *testing.T) {
	items := []appinput.Item{
		{Text: "Item 1", Raw: "item1"},
		{Text: "Item 2", Raw: "item2"},
		{Text: "Item 3", Raw: "item3"},
	}

	list := NewList()
	list.selected = 1 // Select item 2
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	// Create headless window
	sz := image.Point{X: 800, Y: 200}
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

	// Layout with item 1 selected
	list.Layout(gtx, theme, items, make(map[int][]int), false)

	// Frame and screenshot
	hw.Frame(gtx.Ops)
	img := image.NewRGBA(image.Rectangle{Max: sz})
	if err := hw.Screenshot(img); err != nil {
		t.Fatalf("failed to take screenshot: %v", err)
	}

	// Check for blue pixels (selection highlight color: R=0, G=122, B=255)
	// This is a simple check that the highlight is rendered
	foundBlue := false
	for y := 0; y < sz.Y; y++ {
		for x := 0; x < sz.X; x++ {
			r, _, b, _ := img.At(x, y).RGBA()
			// Check for blue-ish color (rough check)
			if r < 100 && b > 200 {
				foundBlue = true
				break
			}
		}
		if foundBlue {
			break
		}
	}

	if !foundBlue {
		t.Error("no blue selection highlight found in rendered list")
	}
}

// TestListNavigationWithFiltering tests navigation after filtering changes item count
func TestListNavigationWithFiltering(t *testing.T) {
	items := []appinput.Item{
		{Text: "Item 1", Raw: "item1"},
		{Text: "Item 2", Raw: "item2"},
	}

	list := NewList()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	var ops op.Ops
	gtx := layout.Context{
		Ops: &ops,
		Metric: unit.Metric{
			PxPerDp: 1.0,
			PxPerSp: 1.0,
		},
		Constraints: layout.Exact(image.Point{X: 800, Y: 600}),
	}

	// Start with 5 items, selection at item 4
	allItems := []appinput.Item{
		{Text: "Item 1", Raw: "item1"},
		{Text: "Item 2", Raw: "item2"},
		{Text: "Item 3", Raw: "item3"},
		{Text: "Item 4", Raw: "item4"},
		{Text: "Item 5", Raw: "item5"},
	}
	list.selected = 4

	// Layout with all items
	list.Layout(gtx, theme, allItems, make(map[int][]int), false)

	if list.selected != 4 {
		t.Errorf("with 5 items, selected = %d, want 4", list.selected)
	}

	// Now filter to only 2 items
	list.Layout(gtx, theme, items, make(map[int][]int), false)

	// Selection should be adjusted to 1 (last valid index)
	if list.selected != 1 {
		t.Errorf("with 2 items after filtering, selected = %d, want 1", list.selected)
	}
}
