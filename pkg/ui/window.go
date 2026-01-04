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
	"github.com/sam33r/goose-launcher/pkg/matcher"
)

// Window manages the launcher UI window
type Window struct {
	app         *app.Window
	theme       *material.Theme
	items       []input.Item      // All items
	filtered    []input.Item      // Filtered items
	list        *List
	searchInput *Input             // Search input field
	matcher     *matcher.FuzzyMatcher // Fuzzy matcher
	selected    string // Selected item (empty if none)
	cancelled   bool   // True if user pressed ESC
}

// NewWindow creates a new launcher window
func NewWindow(items []input.Item) *Window {
	w := new(app.Window)
	w.Option(
		app.Title("Goose Launcher"),
		app.Size(unit.Dp(800), unit.Dp(600)),
	)

	theme := material.NewTheme()

	window := &Window{
		app:         w,
		theme:       theme,
		items:       items,
		filtered:    items, // Initially show all
		list:        NewList(),
		searchInput: NewInput(),
		matcher:     matcher.NewFuzzyMatcher(false, false),
	}

	window.searchInput.Focus()

	return window
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

// filterItems filters items based on the search query
func (w *Window) filterItems(query string) {
	if query == "" {
		w.filtered = w.items
		return
	}

	w.filtered = nil
	for _, item := range w.items {
		if match, _ := w.matcher.Match(query, item); match {
			w.filtered = append(w.filtered, item)
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
		w.list.MoveDown(len(w.filtered)) // Use filtered, not items

	case key.NameReturn, key.NameEnter:
		// Select current item from filtered list
		if len(w.filtered) > 0 {
			idx := w.list.Selected()
			w.selected = w.filtered[idx].Raw
		}

	case key.NameEscape:
		w.cancelled = true
	}
}

// layout renders the window contents
func (w *Window) layout(gtx layout.Context) layout.Dimensions {
	// Register for key events
	event.Op(gtx.Ops, w)

	// Update filtering when input changes
	query := w.searchInput.Text()
	w.filterItems(query)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Search input at top
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return w.searchInput.Layout(gtx, w.theme)
		}),

		// Items list
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return w.list.Layout(gtx, w.theme, w.filtered)
		}),
	)
}
