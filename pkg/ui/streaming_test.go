package ui

import (
	"sync"
	"testing"

	"gioui.org/font/gofont"
	"gioui.org/text"
	"gioui.org/widget/material"

	appinput "github.com/sam33r/goose-launcher/pkg/input"
	"github.com/sam33r/goose-launcher/pkg/matcher"
)

// newStreamingTestWindow builds a Window with the streaming machinery wired
// (pendingItems channel, generation counter) but without a real Gio
// app.Window. Mirrors the literal-init pattern used elsewhere in
// window_test.go so we can drive layout()/filterItems() directly.
func newStreamingTestWindow() *Window {
	w := &Window{
		theme:            material.NewTheme(),
		matchPositions:   make(map[int][]int),
		list:             NewList(),
		searchInput:      NewInput(),
		matcher:          matcher.NewFuzzyMatcher(false, true), // exact mode
		highlightMatches: true,
		pendingItems:     make(chan []appinput.Item, 64),
	}
	w.theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	return w
}

func TestConfigureEmpty_StartsWithNoItems(t *testing.T) {
	w := newStreamingTestWindow()
	// Pretend a previous request left state behind.
	w.items = []appinput.Item{{Text: "leftover"}}
	w.filtered = w.items
	w.selected = "leftover"

	w.ConfigureEmpty(true, true, false)

	if len(w.items) != 0 {
		t.Errorf("ConfigureEmpty: items len = %d, want 0", len(w.items))
	}
	if len(w.filtered) != 0 {
		t.Errorf("ConfigureEmpty: filtered len = %d, want 0", len(w.filtered))
	}
	if w.selected != "" {
		t.Errorf("ConfigureEmpty: selected = %q, want empty", w.selected)
	}
}

func TestAppendItems_ItemsAvailableAfterDrain(t *testing.T) {
	w := newStreamingTestWindow()
	w.ConfigureEmpty(true, true, false)

	first := []appinput.Item{
		mustItem("alpha"),
		mustItem("beta"),
	}
	w.AppendItems(first)

	// Drain (this is what layout() does on every frame).
	w.drainPendingItems()

	if got := len(w.items); got != 2 {
		t.Fatalf("after first append: items len = %d, want 2", got)
	}

	// Filter with empty query — should reflect the appended items.
	w.filterItems("")
	if len(w.filtered) != 2 {
		t.Errorf("filtered len after empty-query filter = %d, want 2", len(w.filtered))
	}

	// Append more.
	w.AppendItems([]appinput.Item{mustItem("gamma")})
	w.drainPendingItems()

	if got := len(w.items); got != 3 {
		t.Errorf("after second append: items len = %d, want 3", got)
	}
}

func TestAppendItems_FilterCacheInvalidatesOnGrowth(t *testing.T) {
	w := newStreamingTestWindow()
	w.ConfigureEmpty(true, true, false)
	w.AppendItems([]appinput.Item{mustItem("apple"), mustItem("banana")})
	w.drainPendingItems()

	// First filter: "a" matches both apple and banana.
	w.filterItems("a")
	if len(w.filtered) != 2 {
		t.Fatalf("first filter: got %d, want 2", len(w.filtered))
	}

	// Append more items under the SAME query — cache must invalidate.
	w.AppendItems([]appinput.Item{mustItem("avocado"), mustItem("kiwi")})
	w.drainPendingItems()

	w.filterItems("a")
	if len(w.filtered) != 3 {
		t.Errorf("after growth + same-query refilter: got %d, want 3 (apple, banana, avocado)", len(w.filtered))
	}
}

func TestAppendItems_FilterCacheStableWhenItemsUnchanged(t *testing.T) {
	// The early-out guard is what keeps idle frames from re-walking 1M items.
	// Verify it still fires when nothing has changed.
	w := newStreamingTestWindow()
	w.ConfigureEmpty(true, true, false)
	w.AppendItems([]appinput.Item{mustItem("apple"), mustItem("apricot")})
	w.drainPendingItems()

	w.filterItems("ap")
	gen1 := w.lastFilteredGeneration
	if gen1 == 0 {
		t.Fatalf("lastFilteredGeneration should be set after first filter, got 0")
	}

	// Re-filter with the same query and no item changes — should be a no-op
	// (generation unchanged, hasFiltered still true).
	w.filterItems("ap")
	if w.lastFilteredGeneration != gen1 {
		t.Errorf("idle re-filter changed generation: %d -> %d", gen1, w.lastFilteredGeneration)
	}
}

func TestAppendItems_ConcurrentProducers(t *testing.T) {
	// Multiple goroutines pushing items at once must not lose any. The drain
	// happens on the consumer (event-loop) goroutine.
	w := newStreamingTestWindow()
	w.ConfigureEmpty(true, true, false)

	const producers = 8
	const perProducer = 25
	var wg sync.WaitGroup
	wg.Add(producers)
	for p := 0; p < producers; p++ {
		go func(pid int) {
			defer wg.Done()
			for i := 0; i < perProducer; i++ {
				w.AppendItems([]appinput.Item{mustItem("p")})
			}
		}(p)
	}
	// Drain in a loop until producers finish + channel emptied.
	doneC := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneC)
	}()
	for {
		w.drainPendingItems()
		select {
		case <-doneC:
			w.drainPendingItems() // final sweep
			goto done
		default:
		}
	}
done:
	want := producers * perProducer
	if got := len(w.items); got != want {
		t.Errorf("concurrent producers: got %d items, want %d", got, want)
	}
}

func mustItem(text string) appinput.Item {
	return appinput.ParseLine(text, 0, "")
}
