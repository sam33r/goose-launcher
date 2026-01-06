package ui

import (
	_ "embed"
	"fmt"
	"image/color"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/font/opentype"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/sam33r/goose-launcher/pkg/input"
	"github.com/sam33r/goose-launcher/pkg/matcher"
)

//go:embed fonts/JetBrainsMono-Regular.ttf
var jetbrainsMonoRegular []byte

//go:embed fonts/JetBrainsMono-Bold.ttf
var jetbrainsMonoBold []byte

// Window manages the launcher UI window
type Window struct {
	app              *app.Window
	theme            *material.Theme
	items            []input.Item      // All items
	filtered         []input.Item      // Filtered items
	matchPositions   map[int][]int     // Mapping of filtered index to match positions
	list             *List
	searchInput      *Input             // Search input field
	matcher          *matcher.FuzzyMatcher // Fuzzy matcher
	selected         string // Selected item (empty if none)
	cancelled        bool   // True if user pressed ESC
	keyTag           bool   // Tag for key events
	highlightMatches bool   // Whether to highlight matching text
}

// NewWindow creates a new launcher window
func NewWindow(items []input.Item, highlightMatches bool, exactMode bool) *Window {
	w := new(app.Window)
	w.Option(
		app.Title("Goose Launcher"),
		app.Size(unit.Dp(800), unit.Dp(600)),
		app.Decorated(false), // Remove OS title bar
	)

	theme := material.NewTheme()

	// Configure JetBrains Mono font
	regular, err := opentype.Parse(jetbrainsMonoRegular)
	if err != nil {
		panic(fmt.Sprintf("failed to parse JetBrains Mono Regular: %v", err))
	}
	bold, err := opentype.Parse(jetbrainsMonoBold)
	if err != nil {
		panic(fmt.Sprintf("failed to parse JetBrains Mono Bold: %v", err))
	}

	collection := []font.FontFace{
		{Font: font.Font{Typeface: "JetBrains Mono"}, Face: regular},
		{Font: font.Font{Typeface: "JetBrains Mono", Weight: font.Bold}, Face: bold},
	}
	theme.Shaper = text.NewShaper(text.WithCollection(collection))

	// fzf-style colors: dark background, light text
	theme.Bg = color.NRGBA{R: 0, G: 0, B: 0, A: 255}           // Black background
	theme.Fg = color.NRGBA{R: 220, G: 220, B: 220, A: 255}     // Light gray text
	theme.ContrastBg = color.NRGBA{R: 30, G: 30, B: 30, A: 255} // Slightly lighter for contrast

	window := &Window{
		app:              w,
		theme:            theme,
		items:            items,
		filtered:         items, // Initially show all
		matchPositions:   make(map[int][]int),
		list:             NewList(),
		searchInput:      NewInput(),
		matcher:          matcher.NewFuzzyMatcher(false, exactMode),
		highlightMatches: highlightMatches,
	}

	window.searchInput.Focus()

	return window
}

// Run starts the window event loop
// Returns selected item or empty string if cancelled
func (w *Window) Run() (string, error) {
	var ops op.Ops

	// Debug: Force window to be visible
	w.app.Option(app.Title("Goose Launcher"))

	for {
		e := w.app.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			return w.selected, e.Err

		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			// Layout the UI
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
		w.matchPositions = make(map[int][]int)
		return
	}

	w.filtered = nil
	w.matchPositions = make(map[int][]int)
	filteredIdx := 0
	for _, item := range w.items {
		match, positions := w.matcher.Match(query, item)
		if match {
			w.filtered = append(w.filtered, item)
			w.matchPositions[filteredIdx] = positions
			filteredIdx++
		}
	}
}

// layout renders the window contents
func (w *Window) layout(gtx layout.Context) layout.Dimensions {
	// Register for keyboard events FIRST (cover entire window area)
	// This ensures window-level keys are registered before editor widget
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	event.Op(gtx.Ops, &w.keyTag)
	area.Pop()

	// Process keyboard events for arrow keys BEFORE rendering
	// This ensures they're handled even if editor has focus
	for {
		ev, ok := gtx.Event(key.Filter{Name: key.NameUpArrow})
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			w.list.MoveUp()
			gtx.Execute(op.InvalidateCmd{})
		}
	}

	for {
		ev, ok := gtx.Event(key.Filter{Name: key.NameDownArrow})
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			w.list.MoveDown(len(w.filtered))
			gtx.Execute(op.InvalidateCmd{})
		}
	}

	// Process Ctrl+J (Down)
	for {
		ev, ok := gtx.Event(key.Filter{Name: "J", Required: key.ModCtrl})
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			w.list.MoveDown(len(w.filtered))
			gtx.Execute(op.InvalidateCmd{})
		}
	}

	// Process Ctrl+K (Up)
	for {
		ev, ok := gtx.Event(key.Filter{Name: "K", Required: key.ModCtrl})
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			w.list.MoveUp()
			gtx.Execute(op.InvalidateCmd{})
		}
	}

	// Process Shift+Return key (for outputting query)
	for {
		ev, ok := gtx.Event(key.Filter{Name: key.NameReturn, Required: key.ModShift})
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			w.selected = w.searchInput.Text()
		}
	}

	// Process Return key (for selection)
	for {
		ev, ok := gtx.Event(key.Filter{Name: key.NameReturn})
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			if e.Modifiers.Contain(key.ModShift) {
				// Shift+Enter: Use current query as selection
				w.selected = w.searchInput.Text()
			} else if len(w.filtered) > 0 {
				// Regular Enter: Select current item from filtered list
				idx := w.list.Selected()
				w.selected = w.filtered[idx].Raw
			}
		}
	}

	// Process ESC key
	for {
		ev, ok := gtx.Event(key.Filter{Name: key.NameEscape})
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			w.cancelled = true
		}
	}

	// Paint dark background (fzf-style)
	paint.Fill(gtx.Ops, w.theme.Bg)

	// Render everything
	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Item count display (fzf-style: "X/Y")
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			countText := fmt.Sprintf("  %d/%d", len(w.filtered), len(w.items))
			label := material.Body1(w.theme, countText)
			label.Color = color.NRGBA{R: 150, G: 150, B: 150, A: 255} // Dim gray
			return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4)}.Layout(gtx, label.Layout)
		}),

		// Search input
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return w.searchInput.Layout(gtx, w.theme)
		}),

		// Items list
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return w.list.Layout(gtx, w.theme, w.filtered, w.matchPositions, w.highlightMatches)
		}),
	)

	// After layout, get current query and update filtering for next frame
	query := w.searchInput.Text()
	w.filterItems(query)

	// Check for item clicks
	clickedIdx := w.list.GetClicked()
	if clickedIdx >= 0 && clickedIdx < len(w.filtered) {
		w.selected = w.filtered[clickedIdx].Raw
		w.list.ResetClicked()
	}

	// Process editor events for text changes
	for {
		ev, ok := w.searchInput.Editor().Update(gtx)
		if !ok {
			break
		}

		// Check for change event (text changed) - request redraw so filtering updates
		if _, ok := ev.(widget.ChangeEvent); ok {
			gtx.Execute(op.InvalidateCmd{})
		}

		// Check for submit event (Enter key from editor)
		if _, ok := ev.(widget.SubmitEvent); ok {
			if w.selected == "" && len(w.filtered) > 0 {
				idx := w.list.Selected()
				w.selected = w.filtered[idx].Raw
			}
		}
	}

	return dims
}
