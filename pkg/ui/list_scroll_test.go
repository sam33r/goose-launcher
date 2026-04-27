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

// TestListMovePageDown_StepsByVisibleCount jumps selection down by one page
// (= number of currently visible rows) and pins the viewport so the new
// selection is visible with the standard scrollOffset of context above it.
func TestListMovePageDown_StepsByVisibleCount(t *testing.T) {
	list := NewList()
	list.list.Position.Count = 10
	list.list.Position.First = 0
	list.selected = 0

	list.MovePageDown(100)

	if list.selected != 10 {
		t.Errorf("selected = %d, want 10 (page=10)", list.selected)
	}
	if !list.needsScroll {
		t.Error("MovePageDown should pin the viewport")
	}
	// targetTop = selected - scrollOffset = 10 - 3 = 7
	if list.scrollToItem != 7 {
		t.Errorf("scrollToItem = %d, want 7 (selected - scrollOffset)", list.scrollToItem)
	}
}

// TestListMovePageDown_ClampsAtEnd never moves past the last item.
func TestListMovePageDown_ClampsAtEnd(t *testing.T) {
	list := NewList()
	list.list.Position.Count = 10
	list.list.Position.First = 80
	list.selected = 95

	list.MovePageDown(100)

	if list.selected != 99 {
		t.Errorf("selected = %d, want 99 (clamped to itemCount-1)", list.selected)
	}
}

// TestListMovePageDown_NoOpAtEnd doesn't trigger a scroll when already at the
// last item — there's nowhere further to go.
func TestListMovePageDown_NoOpAtEnd(t *testing.T) {
	list := NewList()
	list.list.Position.Count = 10
	list.selected = 99

	list.MovePageDown(100)

	if list.selected != 99 {
		t.Errorf("selected = %d, want 99 (already at end)", list.selected)
	}
	if list.needsScroll {
		t.Error("MovePageDown at end should not trigger a scroll")
	}
}

// TestListMovePageUp_StepsByVisibleCount mirrors MovePageDown, going up.
func TestListMovePageUp_StepsByVisibleCount(t *testing.T) {
	list := NewList()
	list.list.Position.Count = 10
	list.list.Position.First = 50
	list.selected = 50

	list.MovePageUp()

	if list.selected != 40 {
		t.Errorf("selected = %d, want 40 (50 - page=10)", list.selected)
	}
	if !list.needsScroll {
		t.Error("MovePageUp should pin the viewport")
	}
	if list.scrollToItem != 37 {
		t.Errorf("scrollToItem = %d, want 37 (selected - scrollOffset)", list.scrollToItem)
	}
}

// TestListMovePageUp_ClampsAtZero never moves below index 0.
func TestListMovePageUp_ClampsAtZero(t *testing.T) {
	list := NewList()
	list.list.Position.Count = 10
	list.selected = 5

	list.MovePageUp()

	if list.selected != 0 {
		t.Errorf("selected = %d, want 0 (clamped to start)", list.selected)
	}
}

// TestListMovePageUp_NoOpAtZero doesn't trigger a scroll when already at index 0.
func TestListMovePageUp_NoOpAtZero(t *testing.T) {
	list := NewList()
	list.list.Position.Count = 10
	list.selected = 0

	list.MovePageUp()

	if list.selected != 0 {
		t.Errorf("selected = %d, want 0 (already at start)", list.selected)
	}
	if list.needsScroll {
		t.Error("MovePageUp at start should not trigger a scroll")
	}
}

// TestListMovePageDown_FallbackWhenNotLayouted falls back to a one-row step
// before the first layout has populated Position.Count, so the binding works
// from the very first frame.
func TestListMovePageDown_FallbackWhenNotLayouted(t *testing.T) {
	list := NewList()
	// Position.Count == 0 (pre-layout default).
	list.selected = 0

	list.MovePageDown(100)

	if list.selected != 1 {
		t.Errorf("selected = %d, want 1 (page-size fallback to 1 when Count==0)", list.selected)
	}
}

// TestApplyWheelDelta_ShiftsSelectionByViewportDelta — the user spun the
// wheel down by 5 rows. Selection moves by the same amount so the cursor's
// position within the visible viewport stays the same.
func TestApplyWheelDelta_ShiftsSelectionByViewportDelta(t *testing.T) {
	list := NewList()
	list.selected = 0
	// After material.List.Layout() returns, Position.First reflects where
	// the wheel landed.
	list.list.Position.First = 5

	list.applyWheelDelta(0, false, 100)

	if list.selected != 5 {
		t.Errorf("selected = %d, want 5 (delta=5)", list.selected)
	}
}

// TestApplyWheelDelta_IgnoresProgrammaticScroll — when WE asked Gio to
// ScrollTo earlier in the frame, the resulting Position.First change is ours,
// not the user's. We must NOT shift selection again.
func TestApplyWheelDelta_IgnoresProgrammaticScroll(t *testing.T) {
	list := NewList()
	list.selected = 8
	list.list.Position.First = 8 // Gio scrolled here because of our ScrollTo

	list.applyWheelDelta(0 /* prevFirst */, true /* didProgrammaticScroll */, 100)

	if list.selected != 8 {
		t.Errorf("selected = %d, want 8 (programmatic scroll must not shift selection)", list.selected)
	}
}

// TestApplyWheelDelta_NoChange — viewport didn't move, selection stays put.
func TestApplyWheelDelta_NoChange(t *testing.T) {
	list := NewList()
	list.selected = 3
	list.list.Position.First = 7

	list.applyWheelDelta(7, false, 100)

	if list.selected != 3 {
		t.Errorf("selected = %d, want 3 (no delta)", list.selected)
	}
}

// TestApplyWheelDelta_ClampsAtEnd — large downward wheel near the end of the
// list clamps selection at itemCount-1, not past it.
func TestApplyWheelDelta_ClampsAtEnd(t *testing.T) {
	list := NewList()
	list.selected = 90
	list.list.Position.First = 95 // wheel jumped down 50

	list.applyWheelDelta(45, false, 100)

	if list.selected != 99 {
		t.Errorf("selected = %d, want 99 (clamped)", list.selected)
	}
}

// TestApplyWheelDelta_ClampsAtZero — upward wheel past the top clamps to 0.
func TestApplyWheelDelta_ClampsAtZero(t *testing.T) {
	list := NewList()
	list.selected = 2
	list.list.Position.First = 0

	list.applyWheelDelta(10, false, 100)

	if list.selected != 0 {
		t.Errorf("selected = %d, want 0 (clamped)", list.selected)
	}
}

// TestApplyWheelDelta_EmptyList is a no-op — guard against itemCount=0.
func TestApplyWheelDelta_EmptyList(t *testing.T) {
	list := NewList()
	list.selected = 0
	list.list.Position.First = 5

	list.applyWheelDelta(0, false, 0)

	// Should not panic; selected stays at 0.
	if list.selected != 0 {
		t.Errorf("selected = %d, want 0 (empty list)", list.selected)
	}
}

// TestApplyWheelDelta_MidViewportTracksOneToOne — when a wheel delta keeps
// selection comfortably inside the viewport (away from both edges), no buffer
// nudge is needed; selection shifts by exactly the wheel delta.
func TestApplyWheelDelta_MidViewportTracksOneToOne(t *testing.T) {
	list := NewList()
	list.list.Position.Count = 20    // visible range after wheel: [50..69]
	list.list.Position.First = 50
	list.selected = 63               // mid-viewport, well inside both buffers

	list.applyWheelDelta(47, false, 100) // delta = 3
	// Naive: 63 + 3 = 66. Buffer band is [53..66]. 66 sits exactly at the
	// bottom edge of the band — no clamp.
	if list.selected != 66 {
		t.Errorf("mid-viewport: selected = %d, want 66 (1:1 tracking)", list.selected)
	}
}

// TestApplyWheelDelta_KeepsBufferBelowTop — when the wheel pushes selection
// up against the top of the visible viewport, nudge it down by scrollOffset
// so the highlight has a few rows of context above it (otherwise the row at
// Position.First can be partially clipped and effectively invisible).
func TestApplyWheelDelta_KeepsBufferBelowTop(t *testing.T) {
	list := NewList()
	list.list.Position.Count = 20
	// Setup: selection was at row 70, viewport was [70..89]. User wheels
	// down 10 rows: viewport becomes [80..99], selection naively lands at
	// 80 (right at the top edge — the visibility bug).
	list.selected = 70
	list.list.Position.First = 80
	list.applyWheelDelta(70, false, 100)
	// With the buffer, expect 80 + scrollOffset = 83.
	if list.selected != 83 {
		t.Errorf("at-top after wheel: selected = %d, want 83 (Position.First + scrollOffset)", list.selected)
	}
}

// TestApplyWheelDelta_NoBufferWhenViewportTooSmall — if the viewport can't
// fit two scrollOffset bands, fall back to plain bounds clamping. Otherwise
// the buffer logic would oscillate or pin selection nonsensically.
func TestApplyWheelDelta_NoBufferWhenViewportTooSmall(t *testing.T) {
	list := NewList()
	list.list.Position.Count = 4 // smaller than 2*scrollOffset (=6)
	list.list.Position.First = 10
	list.selected = 10

	list.applyWheelDelta(5, false, 100) // delta = 5
	// Without buffer interference, expect plain 1:1: 10 + 5 = 15.
	if list.selected != 15 {
		t.Errorf("tiny-viewport: selected = %d, want 15 (no buffer; plain delta)", list.selected)
	}
}

// TestApplyWheelDelta_KeepsBufferAboveBottom — symmetric: wheel that pushes
// selection against the bottom of the viewport should clamp to scrollOffset
// rows above the bottom, so the highlight isn't partially clipped at the
// bottom edge either.
func TestApplyWheelDelta_KeepsBufferAboveBottom(t *testing.T) {
	list := NewList()
	list.list.Position.Count = 20
	// Visible range [11..30]. Selection currently at bottom edge (30).
	// Wheel up by 1 row: prevFirst=12, current First=11, delta=-1.
	// Naive sel = 30 + (-1) = 29, which sits at viewport row [11..30]
	// position 18 — almost at the bottom. Buffer requires sel <=
	// First + Count - 1 - scrollOffset = 11 + 19 - 3 = 27. Expect clamp to 27.
	list.selected = 30
	list.list.Position.First = 11
	list.applyWheelDelta(12, false, 100)
	if list.selected != 27 {
		t.Errorf("at-bottom after wheel: selected = %d, want 27 (bottom - scrollOffset)", list.selected)
	}
}