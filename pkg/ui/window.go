package ui

import (
	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/sam33r/goose-launcher/pkg/input"
)

// Window manages the launcher UI window
type Window struct {
	app       *app.Window
	theme     *material.Theme
	items     []input.Item
	list      *List
	selected  string // Selected item (empty if none)
	cancelled bool   // True if user pressed ESC
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
		list:  NewList(),
	}
}

// Run starts the window event loop
// Returns selected item or empty string if cancelled
func (w *Window) Run() (string, error) {
	var ops op.Ops

	for {
		switch e := w.app.Event().(type) {
		case app.DestroyEvent:
			return w.selected, e.Err

		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			// Handle keyboard events
			for {
				ev, ok := gtx.Event(key.Filter{Focus: w, Name: "", Optional: key.ModShift | key.ModCtrl | key.ModAlt | key.ModSuper})
				if !ok {
					break
				}
				if kev, ok := ev.(key.Event); ok {
					w.handleKey(kev)
				}
			}

			w.layout(gtx)
			e.Frame(&ops)

			// Check if we should close
			if w.selected != "" || w.cancelled {
				return w.selected, nil
			}
		}
	}
}

// handleKey processes keyboard input
func (w *Window) handleKey(e key.Event) {
	if e.State != key.Press {
		return
	}

	switch e.Name {
	case key.NameUpArrow:
		w.list.MoveUp()

	case key.NameDownArrow:
		w.list.MoveDown(len(w.items))

	case key.NameReturn, key.NameEnter:
		// Select current item
		if len(w.items) > 0 {
			idx := w.list.Selected()
			w.selected = w.items[idx].Raw
		}

	case key.NameEscape:
		w.cancelled = true
	}
}

// layout renders the window contents
func (w *Window) layout(gtx layout.Context) layout.Dimensions {
	// Register for key events
	event.Op(gtx.Ops, w)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Items list
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return w.list.Layout(gtx, w.theme, w.items)
		}),
	)
}
