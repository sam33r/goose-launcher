package ui

import (
	"testing"
)

// TestListMoveUp_WithScrollOffset_TriggersScroll tests that MoveUp triggers scroll when above viewport
func TestListMoveUp_WithScrollOffset_TriggersScroll(t *testing.T) {
	list := NewList()
	// Simulate scrolled state (top item is index 10)
	list.list.Position.First = 10
	list.selected = 10

	// MoveUp -> selected becomes 9
	// scrollOffset is 3
	// targetTop = 9 - 3 = 6
	// 6 < 10 (First), so should scroll to 6
	list.MoveUp()

	if list.selected != 9 {
		t.Errorf("expected selected 9, got %d", list.selected)
	}

	if !list.needsScroll {
		t.Error("MoveUp should trigger scroll when target is above viewport")
	}

	expectedScroll := 6
	if list.scrollToItem != expectedScroll {
		t.Errorf("expected scrollToItem %d, got %d", expectedScroll, list.scrollToItem)
	}
}

// TestListMoveUp_NoScrollNeeded tests that MoveUp doesn't scroll if context is visible
func TestListMoveUp_NoScrollNeeded(t *testing.T) {
	list := NewList()
	// Top is 0
	list.list.Position.First = 0
	list.selected = 5

	// MoveUp -> selected 4
	// targetTop = 4 - 3 = 1
	// 1 >= 0. No scroll needed.
	list.MoveUp()

	if list.needsScroll {
		t.Error("MoveUp should not trigger scroll when context is already visible")
	}
}

// TestListMoveDown_WithScrollOffset_TriggersScroll tests that MoveDown triggers scroll when below viewport
func TestListMoveDown_WithScrollOffset_TriggersScroll(t *testing.T) {
	list := NewList()
	// Simulate 10 items visible, starting at 0
	list.list.Position.Count = 10
	list.list.Position.First = 0
	// Visible range: 0 to 9

	list.selected = 6
	// MoveDown -> selected 7
	// scrollOffset = 3
	// targetBottom = 7 + 3 = 10
	// LastVisible = 0 + 10 - 1 = 9
	// 10 > 9. Should scroll.
	
	// Expected NewTop = targetBottom - visibleCount + 2 (conservative)
	// = 10 - 10 + 2 = 2
	
	itemCount := 20
	list.MoveDown(itemCount)

	if list.selected != 7 {
		t.Errorf("expected selected 7, got %d", list.selected)
	}

	if !list.needsScroll {
		t.Error("MoveDown should trigger scroll when target is below viewport")
	}

	expectedScroll := 2
	if list.scrollToItem != expectedScroll {
		t.Errorf("expected scrollToItem %d, got %d", expectedScroll, list.scrollToItem)
	}
}

// TestListMoveDown_FallbackWhenNotLayouted tests fallback behavior when visibleCount is 0
func TestListMoveDown_FallbackWhenNotLayouted(t *testing.T) {
	list := NewList()
	// visibleCount is 0 (default)
	
	list.selected = 5
	itemCount := 20
	
	// MoveDown -> 6
	// targetBottom = 9
	// LastVisible unknown. Fallback triggers.
	
	list.MoveDown(itemCount)
	
	if !list.needsScroll {
		t.Error("MoveDown should trigger scroll (fallback)")
	}
	
	// Fallback scrolls to selected (6) or selected+offset (9) depending on implementation?
	// Implementation: "if l.visibleCount == 0 { l.scrollToItem = l.selected }"
	// So expects 6.
	expectedScroll := 6
	if list.scrollToItem != expectedScroll {
		t.Errorf("expected scrollToItem %d, got %d", expectedScroll, list.scrollToItem)
	}
}