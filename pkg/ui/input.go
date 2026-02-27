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

// Layout renders the input field (fzf-style with ">" prompt)
func (i *Input) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	// Request focus on first layout
	if i.requestFocus {
		gtx.Execute(key.FocusCmd{Tag: &i.editor})
		i.requestFocus = false
	}

	// fzf-style: prompt + input field on dark background
	return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			// ">" prompt (fzf-style)
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				prompt := material.Body1(theme, "> ")
				prompt.Color = color.NRGBA{R: 100, G: 180, B: 255, A: 255} // Blue prompt
				return prompt.Layout(gtx)
			}),

			// Input field
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				editor := material.Editor(theme, &i.editor, "")
				editor.Color = color.NRGBA{R: 220, G: 220, B: 220, A: 255} // Light text
				editor.HintColor = color.NRGBA{R: 100, G: 100, B: 100, A: 255} // Dim hint
				return editor.Layout(gtx)
			}),
		)
	})
}

// Text returns the current input text
func (i *Input) Text() string {
	return i.editor.Text()
}

// SetText sets the input text and moves the cursor to the end
func (i *Input) SetText(text string) {
	i.editor.SetText(text)
	i.editor.SetCaret(len([]rune(text)), len([]rune(text)))
}

// Focus focuses the input field
func (i *Input) Focus() {
	i.requestFocus = true
}

// Editor returns the underlying editor for event handling
func (i *Input) Editor() *widget.Editor {
	return &i.editor
}
