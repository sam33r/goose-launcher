package ui

import (
	"image"
	"testing"

	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// TestInputSetText tests setting and getting text
func TestInputSetText(t *testing.T) {
	inp := NewInput()

	// Initially empty
	if inp.Text() != "" {
		t.Errorf("initial text = %q, want empty", inp.Text())
	}

	// Set text
	inp.SetText("test query")

	if inp.Text() != "test query" {
		t.Errorf("after SetText, text = %q, want %q", inp.Text(), "test query")
	}

	// Change text
	inp.SetText("another query")

	if inp.Text() != "another query" {
		t.Errorf("after second SetText, text = %q, want %q", inp.Text(), "another query")
	}
}

// TestInputFocus tests focus request
func TestInputFocus(t *testing.T) {
	inp := NewInput()

	// Initially not requesting focus
	if inp.requestFocus {
		t.Error("initial requestFocus should be false")
	}

	// Request focus
	inp.Focus()

	if !inp.requestFocus {
		t.Error("after Focus(), requestFocus should be true")
	}

	// Layout should reset it
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	var ops op.Ops
	gtx := layout.Context{
		Ops: &ops,
		Metric: unit.Metric{
			PxPerDp: 1.0,
			PxPerSp: 1.0,
		},
		Constraints: layout.Exact(image.Point{X: 800, Y: 100}),
	}

	inp.Layout(gtx, theme)

	if inp.requestFocus {
		t.Error("after Layout(), requestFocus should be reset to false")
	}
}

// TestInputSubmitEvent tests that Submit is enabled
func TestInputSubmitEvent(t *testing.T) {
	inp := NewInput()

	// Submit should be true (to generate submit events on Enter)
	if !inp.editor.Submit {
		t.Error("editor.Submit should be true to handle Enter key")
	}

	// SingleLine should be true
	if !inp.editor.SingleLine {
		t.Error("editor.SingleLine should be true for search input")
	}
}

// TestInputTextEditing tests text editing via SetText
func TestInputTextEditing(t *testing.T) {
	inp := NewInput()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	var ops op.Ops
	gtx := layout.Context{
		Ops: &ops,
		Metric: unit.Metric{
			PxPerDp: 1.0,
			PxPerSp: 1.0,
		},
		Constraints: layout.Exact(image.Point{X: 800, Y: 100}),
	}

	// Layout once
	inp.Layout(gtx, theme)

	// Set text programmatically
	inp.SetText("hello")

	if inp.Text() != "hello" {
		t.Errorf("after SetText, text = %q, want %q", inp.Text(), "hello")
	}

	// Append more text
	inp.SetText("hello world")

	if inp.Text() != "hello world" {
		t.Errorf("after second SetText, text = %q, want %q", inp.Text(), "hello world")
	}
}

// TestInputChangeEvent tests that text changes work correctly
func TestInputChangeEvent(t *testing.T) {
	inp := NewInput()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	var ops op.Ops
	gtx := layout.Context{
		Ops: &ops,
		Metric: unit.Metric{
			PxPerDp: 1.0,
			PxPerSp: 1.0,
		},
		Constraints: layout.Exact(image.Point{X: 800, Y: 100}),
	}

	// Layout once
	inp.Layout(gtx, theme)

	// Set text programmatically
	inp.SetText("test")

	// Text should be updated
	if inp.Text() != "test" {
		t.Errorf("text = %q, want %q", inp.Text(), "test")
	}

	// Layout again - should not crash
	dims := inp.Layout(gtx, theme)

	if dims.Size.X == 0 || dims.Size.Y == 0 {
		t.Error("layout dimensions should be non-zero")
	}
}

// TestInputEnterKeySubmit tests that Submit is enabled for Enter handling
func TestInputEnterKeySubmit(t *testing.T) {
	inp := NewInput()

	// Submit should be true (to generate submit events on Enter)
	if !inp.editor.Submit {
		t.Error("editor.Submit should be true to handle Enter key properly")
	}

	// Set some text
	inp.SetText("test query")

	if inp.Text() != "test query" {
		t.Errorf("text = %q, want %q", inp.Text(), "test query")
	}

	// Editor configuration should be correct for submit handling
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	var ops op.Ops
	gtx := layout.Context{
		Ops: &ops,
		Metric: unit.Metric{
			PxPerDp: 1.0,
			PxPerSp: 1.0,
		},
		Constraints: layout.Exact(image.Point{X: 800, Y: 100}),
	}

	// Layout should work correctly with Submit enabled
	dims := inp.Layout(gtx, theme)

	if dims.Size.X == 0 || dims.Size.Y == 0 {
		t.Error("layout dimensions should be non-zero")
	}
}

// TestInputDoesNotAddSpaceOnEnter tests that Submit mode prevents newline insertion
func TestInputDoesNotAddSpaceOnEnter(t *testing.T) {
	inp := NewInput()

	// Set initial text
	inp.SetText("search")

	initialText := inp.Text()

	// With Submit: true, Enter should trigger submit event, not add newline/space
	// This is verified by the Submit property being true
	if !inp.editor.Submit {
		t.Error("Submit should be true to prevent Enter from adding text")
	}

	// SingleLine mode also prevents newlines
	if !inp.editor.SingleLine {
		t.Error("SingleLine should be true to prevent newlines")
	}

	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	var ops op.Ops
	gtx := layout.Context{
		Ops: &ops,
		Metric: unit.Metric{
			PxPerDp: 1.0,
			PxPerSp: 1.0,
		},
		Constraints: layout.Exact(image.Point{X: 800, Y: 100}),
	}

	// Layout
	inp.Layout(gtx, theme)

	// Text should remain unchanged
	if inp.Text() != initialText {
		t.Errorf("text changed: got %q, want %q", inp.Text(), initialText)
	}
}

// TestInputLayout tests that layout doesn't panic
func TestInputLayout(t *testing.T) {
	inp := NewInput()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	var ops op.Ops
	gtx := layout.Context{
		Ops: &ops,
		Metric: unit.Metric{
			PxPerDp: 1.0,
			PxPerSp: 1.0,
		},
		Constraints: layout.Exact(image.Point{X: 800, Y: 100}),
	}

	// Should not panic
	dims := inp.Layout(gtx, theme)

	// Should have non-zero dimensions
	if dims.Size.X == 0 || dims.Size.Y == 0 {
		t.Errorf("input dimensions = %v, want non-zero", dims.Size)
	}
}

// TestInputPlaceholder tests that placeholder is shown when empty
func TestInputPlaceholder(t *testing.T) {
	inp := NewInput()

	// Editor should be empty initially
	if inp.Text() != "" {
		t.Errorf("initial text should be empty, got %q", inp.Text())
	}

	// Layout should show placeholder "Search..."
	// This is a visual test - we just ensure layout doesn't crash
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	var ops op.Ops
	gtx := layout.Context{
		Ops: &ops,
		Metric: unit.Metric{
			PxPerDp: 1.0,
			PxPerSp: 1.0,
		},
		Constraints: layout.Exact(image.Point{X: 800, Y: 100}),
	}

	// Should not panic with empty text (placeholder shown)
	dims := inp.Layout(gtx, theme)

	if dims.Size.X == 0 || dims.Size.Y == 0 {
		t.Errorf("input with placeholder dimensions = %v, want non-zero", dims.Size)
	}
}
