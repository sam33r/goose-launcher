package ui

import (
	"image"
	"image/color"
	"strings"

	"gioui.org/font"
	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/sam33r/goose-launcher/pkg/input"
	"github.com/sam33r/goose-launcher/pkg/markup"
)

const scrollOffset = 3 // Keep 3 items context when scrolling

// List displays a scrollable list of items
type List struct {
	list         widget.List
	selected     int
	acceptedIdx  int             // Index of accepted (double-clicked) item (-1 if none)
	clicks       []gesture.Click // One per item; grown lazily in Layout
	scrollToItem int             // Item to scroll to (-1 if no scroll needed)
	needsScroll  bool            // True if we need to scroll on next layout
}

// NewList creates a new list widget
func NewList() *List {
	return &List{
		list: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		selected:     0,
		acceptedIdx:  -1,
		scrollToItem: -1,
		needsScroll:  false,
	}
}

// Layout renders the list
func (l *List) Layout(gtx layout.Context, theme *material.Theme, items []input.Item, matchPositions map[int][]int, highlightMatches bool) layout.Dimensions {
	if len(items) == 0 {
		return layout.Dimensions{}
	}

	// Ensure per-item click gestures cover the current items slice. Grown
	// lazily; never shrunk so streaming-stdin appends don't trigger reallocs.
	if cap(l.clicks) < len(items) {
		grown := make([]gesture.Click, len(items))
		copy(grown, l.clicks)
		l.clicks = grown
	} else if len(l.clicks) < len(items) {
		l.clicks = l.clicks[:len(items)]
	}

	// Ensure selection is in bounds
	if l.selected >= len(items) {
		l.selected = len(items) - 1
	}
	if l.selected < 0 {
		l.selected = 0
	}

	// Snapshot the viewport position before any scroll mutation so we can
	// detect mouse-wheel input by comparing Position.First after layout.
	prevFirst := l.list.Position.First
	didProgrammaticScroll := l.needsScroll && l.scrollToItem >= 0 && l.scrollToItem < len(items)

	// Handle scrolling in layout context
	if didProgrammaticScroll {
		l.list.ScrollTo(l.scrollToItem)
		l.needsScroll = false
		l.scrollToItem = -1
	}

	dims := material.List(theme, &l.list).Layout(gtx, len(items), func(gtx layout.Context, index int) layout.Dimensions {
		matchPos := matchPositions[index]
		return l.layoutItem(gtx, theme, items[index], index, index == l.selected, matchPos, highlightMatches)
	})

	// After layout, Position.First reflects any wheel input from this frame.
	// If we didn't programmatically scroll, attribute the change to the user
	// and shift selection so the highlighted row tracks the viewport.
	l.applyWheelDelta(prevFirst, didProgrammaticScroll, len(items))

	return dims
}

// applyWheelDelta keeps the highlighted row in sync with the viewport when
// the user scrolls the mouse wheel. Called after material.List.Layout()
// returns; Position.First by then reflects any wheel input. When we
// requested a programmatic scroll earlier this frame, the delta is ours and
// must be ignored.
//
// After applying the 1:1 delta, selection is clamped into a "comfortable
// band" inside the viewport — at least scrollOffset rows below the top and
// above the bottom — so the highlight is never partially clipped at either
// edge.
func (l *List) applyWheelDelta(prevFirst int, didProgrammaticScroll bool, itemCount int) {
	if didProgrammaticScroll || itemCount == 0 {
		return
	}
	delta := l.list.Position.First - prevFirst
	if delta == 0 {
		return
	}
	sel := l.selected + delta

	// Keep selection at least scrollOffset rows away from each viewport edge.
	// Skipped when Position.Count isn't populated yet (pre-first-layout) or
	// when the viewport is too small to fit two scrollOffset bands — fall
	// back to plain bounds clamping in those cases.
	if count := l.list.Position.Count; count > 2*scrollOffset {
		first := l.list.Position.First
		topBuffer := first + scrollOffset
		bottomBuffer := first + count - 1 - scrollOffset
		if sel < topBuffer {
			sel = topBuffer
		}
		if sel > bottomBuffer {
			sel = bottomBuffer
		}
	}

	if sel < 0 {
		sel = 0
	}
	if sel >= itemCount {
		sel = itemCount - 1
	}
	l.selected = sel
}

// layoutItem renders a single list item
func (l *List) layoutItem(gtx layout.Context, theme *material.Theme, item input.Item, index int, selected bool, matchPositions []int, highlightMatches bool) layout.Dimensions {
	// Drain pending click events for this item from the previous frame.
	// gesture.Click handles the double-click timing window internally
	// (200 ms in v0.9.0); we just dispatch each KindClick to handleClickEvent.
	if index < len(l.clicks) {
		for {
			ev, ok := l.clicks[index].Update(gtx.Source)
			if !ok {
				break
			}
			l.handleClickEvent(index, ev)
		}
	}

	// Minimal spacing between items
	return layout.UniformInset(unit.Dp(1)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// fzf-style colors
		baseTextColor := color.NRGBA{R: 220, G: 220, B: 220, A: 255}  // Light gray text
		highlightColor := color.NRGBA{R: 255, G: 100, B: 180, A: 255} // Pink/magenta for matches
		selectionBgColor := color.NRGBA{R: 60, G: 60, B: 60, A: 255}  // Lighter gray background for selection

		if selected {
			baseTextColor = color.NRGBA{R: 255, G: 255, B: 255, A: 255} // Pure white when selected
			highlightColor = color.NRGBA{R: 255, G: 180, B: 220, A: 255} // Light pink when selected
		}

		// Use Stack layout pattern for proper vertical centering with minimum height
		return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			minHeight := gtx.Dp(unit.Dp(30))

			// Constrain to minimum height
			gtx.Constraints.Min.Y = minHeight

			dims := layout.Stack{Alignment: layout.W}.Layout(gtx,
				// Layer 1: Full-width background
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					if selected {
						// Use Max.X to span full width, Min.Y for height
						bgSize := gtx.Constraints.Max
						bgSize.Y = gtx.Constraints.Min.Y
						defer clip.Rect{Max: bgSize}.Push(gtx.Ops).Pop()
						paint.Fill(gtx.Ops, selectionBgColor)
						return layout.Dimensions{Size: bgSize}
					}
					return layout.Dimensions{Size: gtx.Constraints.Min}
				}),

				// Layer 2: Text content (Stacked, vertically centered)
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					// Add vertical padding around text
					return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						// Use Center to vertically center the text
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							applyHighlight := highlightMatches && len(matchPositions) > 0
							if applyHighlight || len(item.Spans) > 0 {
								return l.layoutStyledText(gtx, theme, item.Text, item.Spans, matchPositions, applyHighlight, baseTextColor, highlightColor)
							}
							label := material.Body1(theme, item.Text)
							label.Color = baseTextColor
							return label.Layout(gtx)
						})
					})
				}),
			)

			// Register the row's hit area so the per-item gesture.Click
			// receives pointer events. Spans the row's full width and
			// rendered height. Done after layout so dims.Size.Y is known.
			if index < len(l.clicks) {
				rowSize := image.Pt(gtx.Constraints.Max.X, dims.Size.Y)
				area := clip.Rect{Max: rowSize}.Push(gtx.Ops)
				l.clicks[index].Add(gtx.Ops)
				area.Pop()
			}

			return dims
		})
	})
}

// textSegment is a run of characters that share identical rendering attributes.
type textSegment struct {
	content strings.Builder
	color   color.NRGBA
	bold    bool
	italic  bool
	// TODO(markup-underline): add underline bool and draw a 1dp rect under the
	// segment in layoutStyledText when set.
	// TODO(markup-bg): add bg *color.NRGBA and paint a background rect behind
	// the segment when set.
}

// runeAttrs describes per-rune rendering attributes.
type runeAttrs struct {
	color  color.NRGBA
	bold   bool
	italic bool
}

// layoutStyledText renders text that may combine Pango-markup styling (via
// spans) and match highlighting. When both apply to the same rune, the match
// color overrides the span's foreground but bold/italic are preserved.
func (l *List) layoutStyledText(
	gtx layout.Context,
	theme *material.Theme,
	itemText string,
	spans []markup.Span,
	matchPositions []int,
	applyHighlight bool,
	baseColor, highlightColor color.NRGBA,
) layout.Dimensions {
	runes := []rune(itemText)
	if len(runes) == 0 {
		return layout.Dimensions{}
	}

	// Step 1: seed per-rune attributes from spans. If spans are missing or
	// don't cover the full text, fall back to base styling for uncovered runes.
	attrs := make([]runeAttrs, len(runes))
	for i := range attrs {
		attrs[i] = runeAttrs{color: baseColor}
	}
	if len(spans) > 0 {
		cursor := 0
		for _, span := range spans {
			fg := baseColor
			if span.FG != nil {
				fg = *span.FG
			}
			for _, r := range span.Text {
				if cursor >= len(runes) {
					break
				}
				_ = r
				attrs[cursor] = runeAttrs{color: fg, bold: span.Bold, italic: span.Italic}
				cursor++
			}
		}
	}

	// Step 2: overlay match highlighting. Only color changes; bold/italic
	// established by the markup survive so a matched bold char stays bold.
	if applyHighlight {
		for _, pos := range matchPositions {
			if pos >= 0 && pos < len(attrs) {
				attrs[pos].color = highlightColor
			}
		}
	}

	// Step 3: collapse consecutive runes with identical attrs into segments.
	// strings.Builder avoids the O(n²) cost of `cur.content += string(rune)`.
	segments := make([]textSegment, 0, 4)
	segments = append(segments, textSegment{color: attrs[0].color, bold: attrs[0].bold, italic: attrs[0].italic})
	segments[0].content.WriteRune(runes[0])
	for i := 1; i < len(runes); i++ {
		a := attrs[i]
		cur := &segments[len(segments)-1]
		if a.color == cur.color && a.bold == cur.bold && a.italic == cur.italic {
			cur.content.WriteRune(runes[i])
			continue
		}
		segments = append(segments, textSegment{color: a.color, bold: a.bold, italic: a.italic})
		segments[len(segments)-1].content.WriteRune(runes[i])
	}

	// Step 4: render each segment as a labeled flex child with the right font.
	children := make([]layout.FlexChild, len(segments))
	for i := range segments {
		segText := segments[i].content.String()
		segColor := segments[i].color
		segBold := segments[i].bold
		segItalic := segments[i].italic
		children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.Body1(theme, segText)
			label.Color = segColor
			if segBold {
				label.Font.Weight = font.Bold
			}
			if segItalic {
				label.Font.Style = font.Italic
			}
			return label.Layout(gtx)
		})
	}
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}.Layout(gtx, children...)
}

// MoveUp moves selection up
func (l *List) MoveUp() {
	if l.selected > 0 {
		l.selected--
		
		// Ensure context above
		// If selected item is too close to the top edge (or above it), scroll up
		targetTop := l.selected - scrollOffset
		
		// Current first visible item
		firstVisible := l.list.Position.First
		
		// If we are scrolling up past the current view
		if targetTop < firstVisible {
			// Safety: ensure selected is visible at bottom
			// Lowest allowed Top ensures selected is the last visible item
			count := l.list.Position.Count
			if count > 0 {
				minTop := l.selected - count + 1
				if targetTop < minTop {
					targetTop = minTop
				}
			}

			if targetTop < 0 {
				targetTop = 0
			}
			l.scrollToItem = targetTop
			l.needsScroll = true
		}
	}
}

// MoveDown moves selection down
func (l *List) MoveDown(itemCount int) {
	if l.selected < itemCount-1 {
		l.selected++
		
		// Ensure context below
		// We want the selected item + offset to be visible at the bottom
		targetBottom := l.selected + scrollOffset
		
		count := l.list.Position.Count
		
		// Estimate the current last visible item
		// We treat the last item as potentially clipped, so we ignore it for "safe" visibility
		lastSafeVisible := l.list.Position.First + count - 2
		
		if targetBottom > lastSafeVisible {
			// If we haven't rendered yet (count=0), just scroll to selected
			if count == 0 {
				l.scrollToItem = l.selected
			} else {
				// Use +2 to be conservative about partial items
				newTop := targetBottom - count + 2
				
				// Safety: ensure selected is visible at top (don't scroll past selected)
				if newTop > l.selected {
					newTop = l.selected
				}

				if newTop < 0 {
					newTop = 0
				}
				// Don't scroll past end of list
				if newTop >= itemCount {
					newTop = itemCount - 1
				}
				l.scrollToItem = newTop
			}
			l.needsScroll = true
		}
	}
}

// MovePageDown jumps selection down by one page (the number of rows
// currently visible) and pins the viewport so the new selection is visible
// with the standard scrollOffset of context above it. Falls back to a single
// row before the first layout has populated Position.Count.
func (l *List) MovePageDown(itemCount int) {
	page := l.list.Position.Count
	if page <= 0 {
		page = 1
	}
	target := l.selected + page
	if target >= itemCount {
		target = itemCount - 1
	}
	if target <= l.selected {
		return
	}
	l.selected = target

	top := l.selected - scrollOffset
	if top < 0 {
		top = 0
	}
	l.scrollToItem = top
	l.needsScroll = true
}

// MovePageUp mirrors MovePageDown going up.
func (l *List) MovePageUp() {
	page := l.list.Position.Count
	if page <= 0 {
		page = 1
	}
	target := l.selected - page
	if target < 0 {
		target = 0
	}
	if target >= l.selected {
		return
	}
	l.selected = target

	top := l.selected - scrollOffset
	if top < 0 {
		top = 0
	}
	l.scrollToItem = top
	l.needsScroll = true
}

// Selected returns the currently selected index
func (l *List) Selected() int {
	return l.selected
}

// GetAccepted returns the index of an item the user accepted
// (double-clicked) since the last ResetAccepted call, or -1 if none.
// The window layout reads this to write w.selected and complete the request.
func (l *List) GetAccepted() int {
	return l.acceptedIdx
}

// ResetAccepted clears the accepted state. Called by the window layout
// after it has consumed the acceptance.
func (l *List) ResetAccepted() {
	l.acceptedIdx = -1
}

// handleClickEvent routes a single gesture.ClickEvent into the list's
// selection / acceptance state. fzf-style behavior:
//   - single click (NumClicks=1): move highlight to that row, no exit
//   - double click (NumClicks>=2): record the row in acceptedIdx so the
//     window's layout pass exits the request with that selection
//
// Press / Cancel events are ignored — only completed clicks (KindClick)
// move state, otherwise selection would jump on every mouse-down anywhere.
func (l *List) handleClickEvent(idx int, ev gesture.ClickEvent) {
	if ev.Kind != gesture.KindClick {
		return
	}
	l.selected = idx
	if ev.NumClicks >= 2 {
		l.acceptedIdx = idx
	}
}
