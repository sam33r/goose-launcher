package matcher

import (
	"fmt"
	"testing"

	"github.com/sam33r/goose-launcher/pkg/input"
)

// generateItems creates n test items with varying text patterns
func generateItems(n int) []input.Item {
	items := make([]input.Item, n)
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
		items[i] = input.Item{
			Text: text,
			Raw:  text,
		}
	}
	return items
}

// BenchmarkFuzzyMatch_SmallDataset tests matching with 100 items
func BenchmarkFuzzyMatch_SmallDataset(b *testing.B) {
	items := generateItems(100)
	matcher := NewFuzzyMatcher(false, false)
	query := "handler"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, item := range items {
			matcher.Match(query, item)
		}
	}
}

// BenchmarkFuzzyMatch_MediumDataset tests matching with 10k items
func BenchmarkFuzzyMatch_MediumDataset(b *testing.B) {
	items := generateItems(10000)
	matcher := NewFuzzyMatcher(false, false)
	query := "handler"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, item := range items {
			matcher.Match(query, item)
		}
	}
}

// BenchmarkFuzzyMatch_LargeDataset tests matching with 100k items
func BenchmarkFuzzyMatch_LargeDataset(b *testing.B) {
	items := generateItems(100000)
	matcher := NewFuzzyMatcher(false, false)
	query := "handler"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, item := range items {
			matcher.Match(query, item)
		}
	}
}

// BenchmarkFuzzyMatch_VeryLargeDataset tests matching with 1M items
func BenchmarkFuzzyMatch_VeryLargeDataset(b *testing.B) {
	items := generateItems(1000000)
	matcher := NewFuzzyMatcher(false, false)
	query := "handler"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, item := range items {
			matcher.Match(query, item)
		}
	}
}

// BenchmarkFuzzyMatch_ShortQuery tests short queries (1-2 chars)
func BenchmarkFuzzyMatch_ShortQuery(b *testing.B) {
	items := generateItems(10000)
	matcher := NewFuzzyMatcher(false, false)
	query := "h"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, item := range items {
			matcher.Match(query, item)
		}
	}
}

// BenchmarkFuzzyMatch_LongQuery tests long queries (10+ chars)
func BenchmarkFuzzyMatch_LongQuery(b *testing.B) {
	items := generateItems(10000)
	matcher := NewFuzzyMatcher(false, false)
	query := "integration_test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, item := range items {
			matcher.Match(query, item)
		}
	}
}

// BenchmarkFuzzyMatch_WithPositions tests position tracking overhead
func BenchmarkFuzzyMatch_WithPositions(b *testing.B) {
	items := generateItems(10000)
	matcher := NewFuzzyMatcher(false, false)
	query := "handler"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, item := range items {
			matcher.Match(query, item) // Returns positions
		}
	}
}

// BenchmarkExactMatch_MediumDataset tests exact matching with 10k items
func BenchmarkExactMatch_MediumDataset(b *testing.B) {
	items := generateItems(10000)
	matcher := NewFuzzyMatcher(true, false) // exactMode = true
	query := "handler"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, item := range items {
			matcher.Match(query, item)
		}
	}
}

// BenchmarkCaseSensitiveMatch_MediumDataset tests case-sensitive matching
func BenchmarkCaseSensitiveMatch_MediumDataset(b *testing.B) {
	items := generateItems(10000)
	matcher := NewFuzzyMatcher(false, true) // caseSensitive = true
	query := "Handler"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, item := range items {
			matcher.Match(query, item)
		}
	}
}

// BenchmarkMatch_HighMatchRate tests when most items match
func BenchmarkMatch_HighMatchRate(b *testing.B) {
	items := generateItems(10000)
	matcher := NewFuzzyMatcher(false, false)
	query := "go" // Common in most paths

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, item := range items {
			matcher.Match(query, item)
		}
	}
}

// BenchmarkMatch_LowMatchRate tests when few items match
func BenchmarkMatch_LowMatchRate(b *testing.B) {
	items := generateItems(10000)
	matcher := NewFuzzyMatcher(false, false)
	query := "xyzabc" // Unlikely to match

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, item := range items {
			matcher.Match(query, item)
		}
	}
}

// BenchmarkFilterOperation tests full filtering operation (realistic usage)
func BenchmarkFilterOperation(b *testing.B) {
	items := generateItems(10000)
	matcher := NewFuzzyMatcher(false, false)
	query := "handler"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var filtered []input.Item
		var positions [][]int
		for _, item := range items {
			match, pos := matcher.Match(query, item)
			if match {
				filtered = append(filtered, item)
				positions = append(positions, pos)
			}
		}
	}
}
