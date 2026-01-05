package ui

import (
	"image/color"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// Input is a search input field
type Input struct {
	editor        widget.Editor
	requestFocus  bool // True if focus should be requested on next layout
}

// NewInput creates a new input field
func NewInput() *Input {
	return &Input{
		editor: widget.Editor{
			SingleLine: true,
			Submit:     true, // Generate submit event on Enter
		},
		requestFocus: false, // Don't auto-focus - let editor receive events naturally
	}
}

// Layout renders the input field
func (i *Input) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	// Request focus on first layout
	if i.requestFocus {
		gtx.Execute(key.FocusCmd{Tag: &i.editor})
		i.requestFocus = false
	}

	border := widget.Border{
		Color: color.NRGBA{R: 200, G: 200, B: 200, A: 255},
		Width: unit.Dp(1),
	}

	return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				editor := material.Editor(theme, &i.editor, "Search...")
				editor.TextSize = unit.Sp(16)
				return editor.Layout(gtx)
			})
		})
	})
}

// Text returns the current input text
func (i *Input) Text() string {
	return i.editor.Text()
}

// SetText sets the input text
func (i *Input) SetText(text string) {
	i.editor.SetText(text)
}

// Focus focuses the input field
func (i *Input) Focus() {
	i.requestFocus = true
}

// Editor returns the underlying editor for event handling
func (i *Input) Editor() *widget.Editor {
	return &i.editor
}
