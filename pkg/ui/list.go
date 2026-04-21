package ui

import (
	"image/color"

	"gioui.org/font"
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
	list           widget.List
	selected       int
	clickedIdx     int  // Index of clicked item (-1 if none)
	clickTags      []bool // Tags for click tracking (one per potential item)
	scrollToItem   int  // Item to scroll to (-1 if no scroll needed)
	needsScroll    bool // True if we need to scroll on next layout
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
		clickedIdx:   -1,
		clickTags:    make([]bool, 1000), // Pre-allocate tags for up to 1000 items
		scrollToItem: -1,
		needsScroll:  false,
	}
}

// Layout renders the list
func (l *List) Layout(gtx layout.Context, theme *material.Theme, items []input.Item, matchPositions map[int][]int, highlightMatches bool) layout.Dimensions {
	if len(items) == 0 {
		return layout.Dimensions{}
	}

	// DEBUG: Log that we're rendering
	// fmt.Printf("DEBUG: Layout called with %d items\n", len(items))

	// Ensure selection is in bounds
	if l.selected >= len(items) {
		l.selected = len(items) - 1
	}
	if l.selected < 0 {
		l.selected = 0
	}

	// Handle scrolling in layout context
	if l.needsScroll && l.scrollToItem >= 0 && l.scrollToItem < len(items) {
		l.list.ScrollTo(l.scrollToItem)
		l.needsScroll = false
		l.scrollToItem = -1
	}

	return material.List(theme, &l.list).Layout(gtx, len(items), func(gtx layout.Context, index int) layout.Dimensions {
		matchPos := matchPositions[index]
		dims := l.layoutItem(gtx, theme, items[index], index, index == l.selected, matchPos, highlightMatches)
		// DEBUG: Log each item layout
		// fmt.Printf("DEBUG: Item %d layouted, dims=%v\n", index, dims)
		return dims
	})
}

// layoutItem renders a single list item
func (l *List) layoutItem(gtx layout.Context, theme *material.Theme, item input.Item, index int, selected bool, matchPositions []int, highlightMatches bool) layout.Dimensions {
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

			return layout.Stack{Alignment: layout.W}.Layout(gtx,
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
		})
	})
}

// textSegment is a run of characters that share identical rendering attributes.
type textSegment struct {
	content string
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
	var segments []textSegment
	cur := textSegment{content: string(runes[0]), color: attrs[0].color, bold: attrs[0].bold, italic: attrs[0].italic}
	for i := 1; i < len(runes); i++ {
		a := attrs[i]
		if a.color == cur.color && a.bold == cur.bold && a.italic == cur.italic {
			cur.content += string(runes[i])
			continue
		}
		segments = append(segments, cur)
		cur = textSegment{content: string(runes[i]), color: a.color, bold: a.bold, italic: a.italic}
	}
	segments = append(segments, cur)

	// Step 4: render each segment as a labeled flex child with the right font.
	children := make([]layout.FlexChild, len(segments))
	for i, seg := range segments {
		segment := seg
		children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.Body1(theme, segment.content)
			label.Color = segment.color
			if segment.bold {
				label.Font.Weight = font.Bold
			}
			if segment.italic {
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

// Selected returns the currently selected index
func (l *List) Selected() int {
	return l.selected
}

// GetClicked returns the index of the clicked item, or -1 if none
func (l *List) GetClicked() int {
	return l.clickedIdx
}

// ResetClicked resets the clicked state
func (l *List) ResetClicked() {
	l.clickedIdx = -1
}
