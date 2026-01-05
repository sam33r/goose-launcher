package ui

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"

	appinput "github.com/sam33r/goose-launcher/pkg/input"
	"github.com/sam33r/goose-launcher/pkg/matcher"
)

// generateBenchItems creates n test items for benchmarking
func generateBenchItems(n int) []appinput.Item {
	items := make([]appinput.Item, n)
	patterns := []string{
		"user/service/handler/%d",
		"internal/pkg/utils/helper_%d.go",
		"cmd/application/main_%d.go",
		"test/integration/suite_%d_test.go",
		"pkg/model/entity_%d.go",
		"api/v1/endpoint_%d.go",
		"config/environment_%d.yaml",
		"scripts/deployment/deploy_%d.sh",
		"docs/api/reference_%d.md",
		"lib/core/processor_%d.go",
	}

	for i := 0; i < n; i++ {
		pattern := patterns[i%len(patterns)]
		text := fmt.Sprintf(pattern, i)
		items[i] = appinput.Item{
			Text: text,
			Raw:  text,
		}
	}
	return items
}

// setupBenchContext creates a layout context for benchmarking
func setupBenchContext() layout.Context {
	var ops op.Ops
	return layout.Context{
		Ops: &ops,
		Metric: unit.Metric{
			PxPerDp: 1.0,
			PxPerSp: 1.0,
		},
		Constraints: layout.Exact(image.Point{X: 800, Y: 600}),
	}
}

// BenchmarkWindowFilterItems_Small tests filtering with 100 items
func BenchmarkWindowFilterItems_Small(b *testing.B) {
	items := generateBenchItems(100)
	w := &Window{
		theme:            material.NewTheme(),
		items:            items,
		filtered:         items,
		matchPositions:   make(map[int][]int),
		list:             NewList(),
		searchInput:      NewInput(),
		matcher:          matcher.NewFuzzyMatcher(false, false),
		highlightMatches: true,
	}
	w.theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	query := "handler"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.filterItems(query)
	}
}

// BenchmarkWindowFilterItems_Medium tests filtering with 10k items
func BenchmarkWindowFilterItems_Medium(b *testing.B) {
	items := generateBenchItems(10000)
	w := &Window{
		theme:            material.NewTheme(),
		items:            items,
		filtered:         items,
		matchPositions:   make(map[int][]int),
		list:             NewList(),
		searchInput:      NewInput(),
		matcher:          matcher.NewFuzzyMatcher(false, false),
		highlightMatches: true,
	}
	w.theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	query := "handler"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.filterItems(query)
	}
}

// BenchmarkWindowFilterItems_Large tests filtering with 100k items
func BenchmarkWindowFilterItems_Large(b *testing.B) {
	items := generateBenchItems(100000)
	w := &Window{
		theme:            material.NewTheme(),
		items:            items,
		filtered:         items,
		matchPositions:   make(map[int][]int),
		list:             NewList(),
		searchInput:      NewInput(),
		matcher:          matcher.NewFuzzyMatcher(false, false),
		highlightMatches: true,
	}
	w.theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	query := "handler"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.filterItems(query)
	}
}

// BenchmarkWindowFilterItems_VeryLarge tests filtering with 1M items
func BenchmarkWindowFilterItems_VeryLarge(b *testing.B) {
	items := generateBenchItems(1000000)
	w := &Window{
		theme:            material.NewTheme(),
		items:            items,
		filtered:         items,
		matchPositions:   make(map[int][]int),
		list:             NewList(),
		searchInput:      NewInput(),
		matcher:          matcher.NewFuzzyMatcher(false, false),
		highlightMatches: true,
	}
	w.theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	query := "handler"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.filterItems(query)
	}
}

// BenchmarkListLayout_Small tests rendering with 100 items
func BenchmarkListLayout_Small(b *testing.B) {
	items := generateBenchItems(100)
	list := NewList()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	gtx := setupBenchContext()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		list.Layout(gtx, theme, items, make(map[int][]int), false)
	}
}

// BenchmarkListLayout_Medium tests rendering with 1000 items
func BenchmarkListLayout_Medium(b *testing.B) {
	items := generateBenchItems(1000)
	list := NewList()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	gtx := setupBenchContext()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		list.Layout(gtx, theme, items, make(map[int][]int), false)
	}
}

// BenchmarkListLayout_Large tests rendering with 10k items
func BenchmarkListLayout_Large(b *testing.B) {
	items := generateBenchItems(10000)
	list := NewList()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	gtx := setupBenchContext()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		list.Layout(gtx, theme, items, make(map[int][]int), false)
	}
}

// BenchmarkListLayout_WithHighlighting tests rendering overhead with highlighting
func BenchmarkListLayout_WithHighlighting(b *testing.B) {
	items := generateBenchItems(1000)
	list := NewList()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	// Create match positions for all items
	matchPositions := make(map[int][]int)
	for i := 0; i < len(items); i++ {
		matchPositions[i] = []int{0, 1, 2, 3} // Simulate "hand" match
	}

	gtx := setupBenchContext()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		list.Layout(gtx, theme, items, matchPositions, true)
	}
}

// BenchmarkListLayout_WithoutHighlighting tests baseline rendering
func BenchmarkListLayout_WithoutHighlighting(b *testing.B) {
	items := generateBenchItems(1000)
	list := NewList()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	matchPositions := make(map[int][]int)
	for i := 0; i < len(items); i++ {
		matchPositions[i] = []int{0, 1, 2, 3}
	}

	gtx := setupBenchContext()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		list.Layout(gtx, theme, items, matchPositions, false)
	}
}

// BenchmarkHighlightedTextLayout tests text highlighting overhead
func BenchmarkHighlightedTextLayout(b *testing.B) {
	list := NewList()
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	text := "user/service/handler/endpoint.go"
	matchPositions := []int{0, 1, 13, 14, 15, 16, 17, 18, 19} // "us" and "handler"
	baseColor := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	highlightColor := color.NRGBA{R: 255, G: 255, B: 255, A: 255}

	gtx := setupBenchContext()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		list.layoutHighlightedText(gtx, theme, text, matchPositions, baseColor, highlightColor)
	}
}

// BenchmarkWindowLayout_Complete tests complete window layout
func BenchmarkWindowLayout_Complete(b *testing.B) {
	items := generateBenchItems(1000)
	w := &Window{
		theme:            material.NewTheme(),
		items:            items,
		filtered:         items[:100], // Simulate filtered results
		matchPositions:   make(map[int][]int),
		list:             NewList(),
		searchInput:      NewInput(),
		matcher:          matcher.NewFuzzyMatcher(false, false),
		highlightMatches: true,
	}
	w.theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	// Add match positions
	for i := 0; i < 100; i++ {
		w.matchPositions[i] = []int{0, 1, 2}
	}

	gtx := setupBenchContext()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.layout(gtx)
	}
}

// BenchmarkMemoryAllocation_FilterItems tests memory allocation during filtering
func BenchmarkMemoryAllocation_FilterItems(b *testing.B) {
	items := generateBenchItems(10000)
	w := &Window{
		theme:            material.NewTheme(),
		items:            items,
		filtered:         items,
		matchPositions:   make(map[int][]int),
		list:             NewList(),
		searchInput:      NewInput(),
		matcher:          matcher.NewFuzzyMatcher(false, false),
		highlightMatches: true,
	}

	query := "handler"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.filterItems(query)
	}
}

// BenchmarkSearchLatency_Progressive tests progressive search (typing simulation)
func BenchmarkSearchLatency_Progressive(b *testing.B) {
	items := generateBenchItems(100000)
	w := &Window{
		theme:            material.NewTheme(),
		items:            items,
		filtered:         items,
		matchPositions:   make(map[int][]int),
		list:             NewList(),
		searchInput:      NewInput(),
		matcher:          matcher.NewFuzzyMatcher(false, false),
		highlightMatches: true,
	}

	queries := []string{"h", "ha", "han", "hand", "handl", "handle", "handler"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, query := range queries {
			w.filterItems(query)
		}
	}
}
