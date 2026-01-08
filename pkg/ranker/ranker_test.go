package ranker

import (
	"testing"

	"github.com/sam33r/goose-launcher/pkg/input"
)

func TestRankerScoring(t *testing.T) {
	ranker := NewRanker()

	tests := []struct {
		name          string
		query         string
		text          string
		positions     []int
		originalIndex int
		expectHigher  float64 // Minimum expected score
	}{
		{
			name:          "exact match at start",
			query:         "tree",
			text:          "tree.go",
			positions:     []int{0, 1, 2, 3},
			originalIndex: 0,
			expectHigher:  80.0, // High score for perfect match
		},
		{
			name:          "fuzzy match spread out",
			query:         "tree",
			text:          "retrieve_data.go",
			positions:     []int{2, 4, 6, 7},
			originalIndex: 0,
			expectHigher:  30.0, // Lower score for spread match
		},
		{
			name:          "match later in text",
			query:         "tree",
			text:          "src/utils/tree.go",
			positions:     []int{10, 11, 12, 13},
			originalIndex: 0,
			expectHigher:  50.0, // Mid score
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := ranker.scoreMatch(tt.query, tt.text, tt.positions, tt.originalIndex)
			if score < tt.expectHigher {
				t.Errorf("score = %.2f, want >= %.2f", score, tt.expectHigher)
			}
		})
	}
}

func TestRankMatches(t *testing.T) {
	ranker := NewRanker()

	items := []input.Item{
		{Text: "retrieve.go", Raw: "retrieve.go", Index: 0},
		{Text: "tree.go", Raw: "tree.go", Index: 1},
		{Text: "src/tree_utils.go", Raw: "src/tree_utils.go", Index: 2},
	}

	// Positions for query "tree"
	positions := map[int][]int{
		0: {2, 4, 6, 7},   // retrieve - spread out
		1: {0, 1, 2, 3},   // tree - exact match at start
		2: {4, 5, 6, 7},   // tree_utils - exact match but later
	}

	query := "tree"
	scores := ranker.RankMatches(items, positions, query)

	if len(scores) != 3 {
		t.Fatalf("expected 3 scores, got %d", len(scores))
	}

	// The exact match "tree.go" should be ranked first
	if scores[0].Item.Text != "tree.go" {
		t.Errorf("expected 'tree.go' to be ranked first, got '%s'", scores[0].Item.Text)
	}

	// Verify scores are in descending order
	for i := 1; i < len(scores); i++ {
		if scores[i].Score > scores[i-1].Score {
			t.Errorf("scores not in descending order: %.2f > %.2f at index %d",
				scores[i].Score, scores[i-1].Score, i)
		}
	}
}

func TestOriginalPositionWeight(t *testing.T) {
	ranker := NewRanker()

	// Two items with very similar match quality
	items := []input.Item{
		{Text: "tree1.go", Raw: "tree1.go", Index: 0},   // Earlier in list
		{Text: "tree2.go", Raw: "tree2.go", Index: 100}, // Later in list
	}

	// Same match positions (exact match at start for both)
	positions := map[int][]int{
		0: {0, 1, 2, 3},
		1: {0, 1, 2, 3},
	}

	query := "tree"
	scores := ranker.RankMatches(items, positions, query)

	if len(scores) != 2 {
		t.Fatalf("expected 2 scores, got %d", len(scores))
	}

	// The item with lower original index should score higher (tiebreaker)
	if scores[0].OriginalIndex != 0 {
		t.Errorf("expected item at index 0 to rank first, got index %d", scores[0].OriginalIndex)
	}

	// Verify the earlier item has a higher score
	if scores[0].Score <= scores[1].Score {
		t.Errorf("earlier item should have higher score: %.2f <= %.2f",
			scores[0].Score, scores[1].Score)
	}
}

func TestEmptyMatches(t *testing.T) {
	ranker := NewRanker()

	items := []input.Item{}
	positions := map[int][]int{}
	query := "test"

	scores := ranker.RankMatches(items, positions, query)

	if scores != nil {
		t.Errorf("expected nil for empty items, got %v", scores)
	}
}

func TestCompactnessScoring(t *testing.T) {
	ranker := NewRanker()

	tests := []struct {
		name      string
		positions []int
		query     string
		text      string
	}{
		{
			name:      "consecutive chars",
			positions: []int{0, 1, 2, 3},
			query:     "tree",
			text:      "tree.go",
		},
		{
			name:      "spread chars",
			positions: []int{0, 5, 10, 15},
			query:     "tree",
			text:      "t    r    e    e",
		},
	}

	consecutiveScore := ranker.scoreMatch(tests[0].query, tests[0].text, tests[0].positions, 0)
	spreadScore := ranker.scoreMatch(tests[1].query, tests[1].text, tests[1].positions, 0)

	if consecutiveScore <= spreadScore {
		t.Errorf("consecutive match should score higher: %.2f <= %.2f",
			consecutiveScore, spreadScore)
	}
}
