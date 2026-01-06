package ui

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/sam33r/goose-launcher/pkg/input"
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
							if highlightMatches && len(matchPositions) > 0 {
								return l.layoutHighlightedText(gtx, theme, item.Text, matchPositions, baseTextColor, highlightColor)
							} else {
								label := material.Body1(theme, item.Text)
								label.Color = baseTextColor
								return label.Layout(gtx)
							}
						})
					})
				}),
			)
		})
	})
}

// textSegment represents a text segment with a color
type textSegment struct {
	content string
	color   color.NRGBA
}

// layoutHighlightedText renders text with specific characters highlighted
func (l *List) layoutHighlightedText(gtx layout.Context, theme *material.Theme, itemText string, matchPositions []int, baseColor, highlightColor color.NRGBA) layout.Dimensions {
	// Create a set of match positions for quick lookup
	matchSet := make(map[int]bool)
	for _, pos := range matchPositions {
		matchSet[pos] = true
	}

	// Split text into segments with different colors
	var segments []textSegment
	runes := []rune(itemText)
	currentSegment := textSegment{color: baseColor}

	for i, r := range runes {
		isMatch := matchSet[i]
		shouldStartNewSegment := false

		// Check if we need to start a new segment
		if len(currentSegment.content) > 0 {
			// Current segment is highlighted but this char is not, or vice versa
			prevWasMatch := matchSet[i-1]
			if prevWasMatch != isMatch {
				shouldStartNewSegment = true
			}
		}

		if shouldStartNewSegment {
			segments = append(segments, currentSegment)
			if isMatch {
				currentSegment = textSegment{content: string(r), color: highlightColor}
			} else {
				currentSegment = textSegment{content: string(r), color: baseColor}
			}
		} else {
			if isMatch {
				currentSegment.color = highlightColor
			}
			currentSegment.content += string(r)
		}
	}

	// Add the last segment
	if len(currentSegment.content) > 0 {
		segments = append(segments, currentSegment)
	}

	// Layout segments horizontally
	children := make([]layout.FlexChild, len(segments))
	for i, seg := range segments {
		segment := seg // Capture for closure
		children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.Body1(theme, segment.content)
			label.Color = segment.color
			// Don't set TextSize - use theme default for consistency
			return label.Layout(gtx)
		})
	}

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}.Layout(gtx, children...)
}

// MoveUp moves selection up
func (l *List) MoveUp() {
	if l.selected > 0 {
		l.selected--
		// Request scroll to maintain context above
		target := l.selected - scrollOffset
		if target < 0 {
			target = 0
		}
		l.scrollToItem = target
		l.needsScroll = true
	}
}

// MoveDown moves selection down
func (l *List) MoveDown(itemCount int) {
	if l.selected < itemCount-1 {
		l.selected++
		// Request scroll to maintain context below
		target := l.selected + scrollOffset
		if target >= itemCount {
			target = itemCount - 1
		}
		l.scrollToItem = target
		l.needsScroll = true
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
