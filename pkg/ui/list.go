package ui

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/sam33r/goose-launcher/pkg/input"
)

// List displays a scrollable list of items
type List struct {
	list     widget.List
	selected int
}

// NewList creates a new list widget
func NewList() *List {
	return &List{
		list: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		selected: 0,
	}
}

// Layout renders the list
func (l *List) Layout(gtx layout.Context, theme *material.Theme, items []input.Item) layout.Dimensions {
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
		return l.layoutItem(gtx, theme, items[index], index == l.selected)
	})
}

// layoutItem renders a single list item
func (l *List) layoutItem(gtx layout.Context, theme *material.Theme, item input.Item, selected bool) layout.Dimensions {
	// Background color for selected item
	if selected {
		// Draw selection background
		// (simplified - full implementation would use paint.ColorOp)
	}

	// Display item text
	label := material.Body1(theme, item.Text)

	if selected {
		label.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255} // White text
	} else {
		label.Color = color.NRGBA{R: 0, G: 0, B: 0, A: 255} // Black text
	}

	return layout.UniformInset(unit.Dp(8)).Layout(gtx, label.Layout)
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
