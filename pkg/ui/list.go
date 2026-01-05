package ui

import (
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
	list       widget.List
	selected   int
	clickedIdx int  // Index of clicked item (-1 if none)
	clickTags  []bool // Tags for click tracking (one per potential item)
}

// NewList creates a new list widget
func NewList() *List {
	return &List{
		list: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		selected:   0,
		clickedIdx: -1,
		clickTags:  make([]bool, 1000), // Pre-allocate tags for up to 1000 items
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

	return material.List(theme, &l.list).Layout(gtx, len(items), func(gtx layout.Context, index int) layout.Dimensions {
		matchPos := matchPositions[index]
		return l.layoutItem(gtx, theme, items[index], index, index == l.selected, matchPos, highlightMatches)
	})
}

// layoutItem renders a single list item
func (l *List) layoutItem(gtx layout.Context, theme *material.Theme, item input.Item, index int, selected bool, matchPositions []int, highlightMatches bool) layout.Dimensions {
	return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// Determine text color based on selection
		baseTextColor := color.NRGBA{R: 0, G: 0, B: 0, A: 255}       // Black text
		highlightColor := color.NRGBA{R: 255, G: 0, B: 0, A: 255}    // Red highlight
		if selected {
			baseTextColor = color.NRGBA{R: 255, G: 255, B: 255, A: 255} // White text when selected
			highlightColor = color.NRGBA{R: 255, G: 200, B: 200, A: 255} // Light red highlight when selected
		}

		// Layout the text (with or without highlighting)
		var textDims layout.Dimensions
		macro := op.Record(gtx.Ops)

		if highlightMatches && len(matchPositions) > 0 {
			// Render with highlighting
			textDims = l.layoutHighlightedText(gtx, theme, item.Text, matchPositions, baseTextColor, highlightColor)
		} else {
			// Render without highlighting
			label := material.Body1(theme, item.Text)
			label.Color = baseTextColor
			textDims = label.Layout(gtx)
		}

		call := macro.Stop()

		// Draw selection background
		if selected {
			bgColor := color.NRGBA{R: 0, G: 122, B: 255, A: 255}
			rect := clip.Rect{Max: textDims.Size}.Push(gtx.Ops)
			paint.Fill(gtx.Ops, bgColor)
			rect.Pop()
		}

		// Register click area
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

		// Draw the text on top
		call.Add(gtx.Ops)

		return textDims
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
			label.TextSize = unit.Sp(14)
			return label.Layout(gtx)
		})
	}

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}.Layout(gtx, children...)
}

// MoveUp moves selection up
func (l *List) MoveUp() {
	if l.selected > 0 {
		l.selected--
	}
}

// MoveDown moves selection down
func (l *List) MoveDown(itemCount int) {
	if l.selected < itemCount-1 {
		l.selected++
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
