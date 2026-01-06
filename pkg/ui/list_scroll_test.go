package ui

import (
	"testing"
)

// TestListMoveUp_WithScrollOffset tests that MoveUp triggers scrolling with offset
func TestListMoveUp_WithScrollOffset(t *testing.T) {
	list := NewList()

	// Setup a list with 10 items
	// Assume scrollOffset is 3 (defined in constants)

	// Start at index 5
	list.selected = 5

	// Move up
	list.MoveUp()

	// Selected should decrease
	if list.selected != 4 {
		t.Errorf("expected selected 4, got %d", list.selected)
	}

	// Should trigger scroll to maintain context above
	// If offset is 3, moving to 4 should try to make 4-3 = 1 visible
	if !list.needsScroll {
		t.Error("MoveUp should trigger scroll")
	}

	expectedScroll := 1 // 4 - 3 = 1
	if list.scrollToItem != expectedScroll {
		t.Errorf("expected scrollToItem %d, got %d", expectedScroll, list.scrollToItem)
	}

	// Move up again to 3
	list.MoveUp()
	expectedScroll = 0 // 3 - 3 = 0
	if list.scrollToItem != expectedScroll {
		t.Errorf("expected scrollToItem %d, got %d", expectedScroll, list.scrollToItem)
	}

	// Move up near top (index 1)
	list.selected = 1
	list.MoveUp() // to 0
	expectedScroll = 0 // 0 - 3 < 0, capped at 0
	if list.scrollToItem != expectedScroll {
		t.Errorf("expected scrollToItem %d (capped at 0), got %d", expectedScroll, list.scrollToItem)
	}
}

// TestListMoveDown_WithScrollOffset tests that MoveDown triggers scrolling with offset
func TestListMoveDown_WithScrollOffset(t *testing.T) {
	list := NewList()
	itemCount := 20

	// Start at index 10
	list.selected = 10

	// Move down
	list.MoveDown(itemCount)

	// Selected should increase
	if list.selected != 11 {
		t.Errorf("expected selected 11, got %d", list.selected)
	}

	// Should trigger scroll to maintain context below
	// If offset is 3, moving to 11 should try to make 11+3 = 14 visible
	if !list.needsScroll {
		t.Error("MoveDown should trigger scroll")
	}

	expectedScroll := 14 // 11 + 3 = 14
	if list.scrollToItem != expectedScroll {
		t.Errorf("expected scrollToItem %d, got %d", expectedScroll, list.scrollToItem)
	}

	// Move down near bottom (index 18 of 20)
	list.selected = 18
	list.MoveDown(itemCount) // to 19 (last item)
	
	expectedScroll = 19 // 19 + 3 > 19, capped at last item
	if list.scrollToItem != expectedScroll {
		t.Errorf("expected scrollToItem %d (capped at last), got %d", expectedScroll, list.scrollToItem)
	}
}
