package ui

import (
	"image"
	"image/color"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/sam33r/goose-launcher/pkg/input"
)

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
		return l.layoutItem(gtx, theme, items[index], index, index == l.selected, matchPos, highlightMatches)
	})
}

// layoutItem renders a single list item
func (l *List) layoutItem(gtx layout.Context, theme *material.Theme, item input.Item, index int, selected bool, matchPositions []int, highlightMatches bool) layout.Dimensions {
	return layout.UniformInset(unit.Dp(2)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// fzf-style colors: light text on dark background
		baseTextColor := color.NRGBA{R: 220, G: 220, B: 220, A: 255}  // Light gray text
		highlightColor := color.NRGBA{R: 255, G: 100, B: 180, A: 255} // Pink/magenta for matches
		barColor := color.NRGBA{R: 255, G: 0, B: 128, A: 255}         // Pink/magenta bar
		barWidth := gtx.Dp(unit.Dp(4))
		barPadding := gtx.Dp(unit.Dp(8)) // Padding between bar and text

		if selected {
			baseTextColor = color.NRGBA{R: 255, G: 255, B: 255, A: 255} // Pure white when selected
			highlightColor = color.NRGBA{R: 255, G: 180, B: 220, A: 255} // Light pink when selected
		}

		// Use Flex layout to properly position bar and text
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			// Selection bar (if selected)
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !selected {
					// Reserve space even when not selected for consistent alignment
					return layout.Dimensions{Size: image.Pt(barWidth+barPadding, 0)}
				}

				// Draw selection bar
				barSize := image.Pt(barWidth, gtx.Constraints.Max.Y)
				defer clip.Rect{Max: barSize}.Push(gtx.Ops).Pop()
				paint.Fill(gtx.Ops, barColor)

				return layout.Dimensions{
					Size: image.Pt(barWidth+barPadding, gtx.Constraints.Max.Y),
				}
			}),

			// Text content (render directly without background for now)
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				// Render text directly
				var textDims layout.Dimensions
				if highlightMatches && len(matchPositions) > 0 {
					textDims = l.layoutHighlightedText(gtx, theme, item.Text, matchPositions, baseTextColor, highlightColor)
				} else {
					label := material.Body1(theme, item.Text)
					label.Color = baseTextColor
					textDims = label.Layout(gtx)
				}

				// Set up click handling
				clickArea := clip.Rect{Max: textDims.Size}.Push(gtx.Ops)
				event.Op(gtx.Ops, &l.clickTags[index])

				// Check for clicks
				for {
					ev, ok := gtx.Event(pointer.Filter{
						Target: &l.clickTags[index],
						Kinds:  pointer.Press,
					})
					if !ok {
						break
					}
					if _, ok := ev.(pointer.Event); ok {
						l.clickedIdx = index
						l.selected = index
						gtx.Execute(op.InvalidateCmd{})
					}
				}
				clickArea.Pop()

				return textDims
			}),
		)
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
		// Request scroll to make selected item visible (will happen on next layout)
		l.scrollToItem = l.selected
		l.needsScroll = true
	}
}

// MoveDown moves selection down
func (l *List) MoveDown(itemCount int) {
	if l.selected < itemCount-1 {
		l.selected++
		// Request scroll to make selected item visible (will happen on next layout)
		l.scrollToItem = l.selected
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
