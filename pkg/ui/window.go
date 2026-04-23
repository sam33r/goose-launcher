package ui

import (
	_ "embed"
	"fmt"
	"image/color"
	"time"

	"gioui.org/app"
	"gioui.org/font"
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

	"github.com/sam33r/goose-launcher/pkg/fontcache"
	"github.com/sam33r/goose-launcher/pkg/input"
	"github.com/sam33r/goose-launcher/pkg/matcher"
	"github.com/sam33r/goose-launcher/pkg/ranker"
)

//go:embed fonts/JetBrainsMono-Regular.ttf
var jetbrainsMonoRegular []byte

//go:embed fonts/JetBrainsMono-Bold.ttf
var jetbrainsMonoBold []byte

//go:embed fonts/JetBrainsMono-Italic.ttf
var jetbrainsMonoItalic []byte

// Window manages the launcher UI window.
//
// Lifecycle modes:
//
//   - Standalone (legacy): NewWindow(items, ...) → Run() → exit.
//     Window is created visible; Run() returns on selection/destroy.
//   - Daemon (phase 2): NewWindowEmpty() → go RunForever() →
//     WaitForFirstFrame() → loop {Configure(items,...) → external show →
//     WaitForSelection() → external hide}. The same *Window object is
//     reused across many user invocations.
//
// Per-request state (items, filtered, selected, etc.) is reset by
// Configure. Persistent state (app.Window, theme, widgets) survives.
type Window struct {
	app              *app.Window
	theme            *material.Theme
	items            []input.Item      // All items
	filtered         []input.Item      // Filtered items
	matchPositions   map[int][]int     // Mapping of filtered index to match positions
	list             *List
	searchInput      *Input             // Search input field
	matcher          *matcher.FuzzyMatcher // Fuzzy matcher
	ranker           *ranker.Ranker        // Match ranker/scorer
	rankEnabled      bool                  // Whether to rank results
	selected         string // Selected item (empty if none)
	cancelled        bool   // True if user pressed ESC
	keyTag           bool   // Tag for key events
	highlightMatches bool   // Whether to highlight matching text
	metrics          StartupMetrics // Startup performance metrics
	firstFrame       bool           // Track if first frame rendered

	// lastQuery is the query the current filtered/matchPositions reflect.
	// Layout runs filter on every frame; without this guard we re-filter the
	// whole input on idle frames (mouse move, focus events) — at 1M items
	// that costs ~140 ms per redraw.
	lastQuery   string
	hasFiltered bool
	// filteredOwned is the backing slice we control. w.filtered may alias
	// w.items when the query is empty; we keep filteredOwned separate so
	// subsequent non-empty queries don't write through into w.items.
	filteredOwned []input.Item

	// Daemon-mode signaling. nil channels are fine (no daemon waiting); the
	// non-blocking sends elsewhere handle that case.
	requestDone    chan struct{} // closed when current request completes (selection or cancel)
	firstFrameOnce chan struct{} // closed exactly once after the first FrameEvent
}

// NewWindow creates a launcher window pre-loaded with items. Standalone
// (legacy) entry point — kept for tests. Daemon callers should use
// NewWindowEmpty + Configure.
func NewWindow(items []input.Item, highlightMatches bool, exactMode bool, rankEnabled bool) *Window {
	w := newWindowShell()
	w.Configure(items, highlightMatches, exactMode, rankEnabled)
	return w
}

// NewWindowEmpty creates the persistent daemon window with no items
// loaded. The caller must invoke Configure before each request.
func NewWindowEmpty() *Window {
	return newWindowShell()
}

// newWindowShell handles the one-time setup that's identical for both
// constructors: app.Window, theme, fonts, persistent widgets.
func newWindowShell() *Window {
	var metrics StartupMetrics
	if BenchmarkMode {
		metrics.WindowCreationStart = time.Now()
	}

	w := new(app.Window)
	w.Option(
		app.Title("Goose Launcher"),
		app.Decorated(false), // Remove OS title bar
		app.Maximized.Option(),
	)

	theme := material.NewTheme()

	// Configure JetBrains Mono font (using cache)
	regular, bold, italic, err := fontcache.GetFonts(jetbrainsMonoRegular, jetbrainsMonoBold, jetbrainsMonoItalic)
	if err != nil {
		panic(fmt.Sprintf("failed to load fonts: %v", err))
	}

	collection := []font.FontFace{
		{Font: font.Font{Typeface: "JetBrains Mono"}, Face: regular},
		{Font: font.Font{Typeface: "JetBrains Mono", Weight: font.Bold}, Face: bold},
		{Font: font.Font{Typeface: "JetBrains Mono", Style: font.Italic}, Face: italic},
	}
	theme.Shaper = text.NewShaper(text.WithCollection(collection))

	// fzf-style colors: dark background, light text
	theme.Bg = color.NRGBA{R: 0, G: 0, B: 0, A: 255}           // Black background
	theme.Fg = color.NRGBA{R: 220, G: 220, B: 220, A: 255}     // Light gray text
	theme.ContrastBg = color.NRGBA{R: 30, G: 30, B: 30, A: 255} // Slightly lighter for contrast

	window := &Window{
		app:            w,
		theme:          theme,
		matchPositions: make(map[int][]int),
		list:           NewList(),
		searchInput:    NewInput(),
		ranker:         ranker.NewRanker(),
		metrics:        metrics,
		firstFrame:     true,
		firstFrameOnce: make(chan struct{}),
	}
	window.searchInput.Focus()

	if BenchmarkMode {
		window.metrics.WindowCreationEnd = time.Now()
	}
	return window
}

// Configure prepares the window for a new request. Resets per-request state
// (items, filter cache, selection, list cursor, search input) without
// touching the persistent app.Window/theme. Call once between
// WaitForSelection returns and the next external show.
//
// Daemon callers must not call Configure while the event loop is mid-frame
// for a previous request — the daemon's serialization (workMu) ensures this
// by draining requests one-at-a-time.
func (w *Window) Configure(items []input.Item, highlightMatches, exactMode, rankEnabled bool) {
	w.items = items
	w.filtered = items
	for k := range w.matchPositions {
		delete(w.matchPositions, k)
	}
	w.matcher = matcher.NewFuzzyMatcher(false, exactMode)
	w.rankEnabled = rankEnabled
	w.highlightMatches = highlightMatches

	// Reset per-request runtime state.
	w.selected = ""
	w.cancelled = false
	w.lastQuery = ""
	w.hasFiltered = false
	w.filteredOwned = w.filteredOwned[:0]

	// Reset list cursor + clicked tracker. List has no public reset, but
	// MoveUp from index 0 is a no-op so we synthesize it.
	w.list.selected = 0
	w.list.clickedIdx = -1
	w.list.scrollToItem = 0
	w.list.needsScroll = true

	// Clear the search input.
	w.searchInput.SetText("")

	// Fresh per-request signal channel. Closed when selection/cancel happens.
	w.requestDone = make(chan struct{})
}

// GioWindow exposes the underlying *app.Window so callers can call
// Invalidate() to wake the event loop after externally showing the window.
func (w *Window) GioWindow() *app.Window { return w.app }

// GetMetrics returns the startup metrics for this window
func (w *Window) GetMetrics() StartupMetrics {
	return w.metrics
}

// SetEarlyMetrics seeds the pre-window timestamps that main() captured.
// Any zero time is left untouched so partial data (e.g. no LAUNCH_START_NS)
// is fine.
func (w *Window) SetEarlyMetrics(launch, proc, stdinStart, stdinEnd time.Time) {
	if !launch.IsZero() {
		w.metrics.LaunchStart = launch
	}
	if !proc.IsZero() {
		w.metrics.ProcessStart = proc
	}
	if !stdinStart.IsZero() {
		w.metrics.StdinReadStart = stdinStart
	}
	if !stdinEnd.IsZero() {
		w.metrics.StdinReadEnd = stdinEnd
	}
}

// Run drives one request to completion. Returns selected item (empty on
// cancel) when the user picks or dismisses, or returns on DestroyEvent.
// Standalone (legacy) entry point — daemon callers use RunForever.
func (w *Window) Run() (string, error) {
	var ops op.Ops
	w.app.Option(app.Title("Goose Launcher"))

	for {
		e := w.app.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			return w.selected, e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			w.layout(gtx)
			e.Frame(&ops)
			w.markFirstFrameDone()
			if w.selected != "" || w.cancelled {
				return w.selected, nil
			}
		}
	}
}

// RunForever pumps events for the lifetime of the daemon. On each request,
// the daemon calls Configure to reset state and waits on WaitForSelection.
// This loop signals completion by closing w.requestDone (set in Configure)
// when selected/cancelled becomes true. Returns only when the OS destroys
// the window.
func (w *Window) RunForever() error {
	var ops op.Ops

	for {
		e := w.app.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			w.layout(gtx)
			e.Frame(&ops)
			w.markFirstFrameDone()

			// Once a request completes, signal the daemon's request handler
			// (which is parked in WaitForSelection). We close the channel so
			// concurrent waiters all unblock; the daemon installs a fresh
			// channel in Configure for the next request.
			if (w.selected != "" || w.cancelled) && w.requestDone != nil {
				close(w.requestDone)
				w.requestDone = nil
			}
		}
	}
}

// markFirstFrameDone closes firstFrameOnce exactly once. Used by daemon
// startup to know when the NSWindow exists so it can locate the pointer
// via [NSApp windows].
func (w *Window) markFirstFrameDone() {
	if BenchmarkMode && w.firstFrame {
		w.metrics.FirstFrameTime = time.Now()
	}
	if w.firstFrame {
		w.firstFrame = false
		if w.firstFrameOnce != nil {
			close(w.firstFrameOnce)
		}
	}
}

// WaitForFirstFrame blocks until the first FrameEvent has been processed.
// Used by the daemon to know when Gio's NSWindow has been created.
// Idempotent — returns immediately on subsequent calls.
func (w *Window) WaitForFirstFrame() {
	if w.firstFrameOnce == nil {
		return
	}
	<-w.firstFrameOnce
}

// WaitForSelection blocks until the current request completes. Returns the
// selected item (empty on cancel/dismiss). Must be preceded by a Configure
// call that installed a fresh requestDone channel.
func (w *Window) WaitForSelection() string {
	if w.requestDone != nil {
		<-w.requestDone
	}
	return w.selected
}

// Cancel dismisses the current request as if the user pressed ESC. Used by
// the daemon's benchmark path to measure show→done latency without human
// interaction. Wakes Run/RunForever via Invalidate so the cancelled flag is
// observed promptly.
func (w *Window) Cancel() {
	w.cancelled = true
	if w.app != nil {
		w.app.Invalidate()
	}
}

// filterItems filters items based on the search query.
// No-op when called repeatedly with the same query (the layout pass calls this
// on every frame; we don't want to re-walk a million items on idle redraws).
func (w *Window) filterItems(query string) {
	if w.hasFiltered && query == w.lastQuery {
		return
	}
	w.lastQuery = query
	w.hasFiltered = true

	if query == "" {
		w.filtered = w.items
		// Reuse the existing map allocation when possible to avoid GC churn.
		for k := range w.matchPositions {
			delete(w.matchPositions, k)
		}
		return
	}

	// Whether downstream consumers actually need positions; skipping the
	// allocation cuts ~1 alloc/match for the --highlight-matches=false path.
	needPositions := w.highlightMatches || w.rankEnabled

	// Reuse the filtered slice's backing array across frames so progressive
	// typing doesn't reallocate. Always go through filteredOwned — we never
	// want to write through w.filtered when it's aliased to w.items.
	filtered := w.filteredOwned[:0]

	// Reuse the positions map; clearing is cheaper than a fresh allocation
	// for the common case where the result-set size doesn't change much.
	for k := range w.matchPositions {
		delete(w.matchPositions, k)
	}

	filteredIdx := 0
	for _, item := range w.items {
		var (
			match     bool
			positions []int
		)
		if needPositions {
			match, positions = w.matcher.Match(query, item)
		} else {
			match = w.matcher.MatchOnly(query, item)
		}
		if match {
			filtered = append(filtered, item)
			if needPositions {
				w.matchPositions[filteredIdx] = positions
			}
			filteredIdx++
		}
	}
	w.filteredOwned = filtered
	w.filtered = filtered

	// Optional ranking pass.
	if w.rankEnabled && len(w.filtered) > 0 {
		scores := w.ranker.RankMatches(w.filtered, w.matchPositions, query)
		filtered = filtered[:0]
		for k := range w.matchPositions {
			delete(w.matchPositions, k)
		}
		for i, score := range scores {
			filtered = append(filtered, score.Item)
			if needPositions {
				w.matchPositions[i] = score.Positions
			}
		}
		w.filteredOwned = filtered
		w.filtered = filtered
	}
}

// layout renders the window contents
func (w *Window) layout(gtx layout.Context) layout.Dimensions {
	// Track first layout timing
	if BenchmarkMode && w.firstFrame && w.metrics.FirstLayoutTime.IsZero() {
		w.metrics.FirstLayoutTime = time.Now()
	}

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
			} else if w.searchInput.Text() != "" {
				// No matches but text in input: output the query text (like Shift+Enter)
				w.selected = w.searchInput.Text()
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

	// Process Tab key (fill input with selected item's raw text)
	for {
		ev, ok := gtx.Event(key.Filter{Name: key.NameTab})
		if !ok {
			break
		}
		if e, ok := ev.(key.Event); ok && e.State == key.Press {
			if len(w.filtered) > 0 {
				idx := w.list.Selected()
				w.searchInput.SetText(w.filtered[idx].Raw)
				gtx.Execute(op.InvalidateCmd{})
			}
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
			} else if w.selected == "" && w.searchInput.Text() != "" {
				w.selected = w.searchInput.Text()
			}
		}
	}

	return dims
}
