package ui

import (
	"fmt"
	"image/color"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/sam33r/goose-launcher/pkg/input"
)

// Window manages the launcher UI window
type Window struct {
	app   *app.Window
	theme *material.Theme
	items []input.Item
}

// NewWindow creates a new launcher window
func NewWindow(items []input.Item) *Window {
	w := new(app.Window)
	w.Option(
		app.Title("Goose Launcher"),
		app.Size(unit.Dp(800), unit.Dp(600)),
	)

	theme := material.NewTheme()

	return &Window{
		app:   w,
		theme: theme,
		items: items,
	}
}

// Run starts the window event loop
// Returns selected item or empty string if cancelled
func (w *Window) Run() (string, error) {
	var ops op.Ops

	for {
		switch e := w.app.Event().(type) {
		case app.DestroyEvent:
			return "", e.Err

		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			w.layout(gtx)
			e.Frame(&ops)
		}
	}
}

// layout renders the window contents
func (w *Window) layout(gtx layout.Context) layout.Dimensions {
	// Simple layout: just show item count for now
	text := fmt.Sprintf("%d items loaded", len(w.items))

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		label := material.H1(w.theme, text)
		label.Color = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
		return label.Layout(gtx)
	})
}
