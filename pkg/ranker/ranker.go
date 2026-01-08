package ranker

import (
	"sort"

	"github.com/sam33r/goose-launcher/pkg/input"
)

// MatchScore represents a scored match result
type MatchScore struct {
	Item            input.Item
	Score           float64
	Positions       []int
	OriginalIndex   int // Position in original input list
}

// Ranker handles scoring and ranking of matched items
type Ranker struct {
	// Weights for different scoring components (sum to 100)
	CompactnessWeight   float64 // How close together the matched chars are
	EarlyMatchWeight    float64 // Bonus for matches at start of text
	ConsecutiveWeight   float64 // Bonus for consecutive character matches
	LengthRatioWeight   float64 // Query length vs text length
	OriginalPosWeight   float64 // Preference for items earlier in original list
}

// NewRanker creates a ranker with default weights
func NewRanker() *Ranker {
	return &Ranker{
		CompactnessWeight:   35.0,
		EarlyMatchWeight:    25.0,
		ConsecutiveWeight:   20.0,
		LengthRatioWeight:   10.0,
		OriginalPosWeight:   10.0,
	}
}

// RankMatches scores and sorts matched items by relevance
// Returns sorted slice of MatchScore
func (r *Ranker) RankMatches(items []input.Item, positions map[int][]int, query string) []MatchScore {
	if len(items) == 0 {
		return nil
	}

	scores := make([]MatchScore, 0, len(items))

	for i, item := range items {
		pos := positions[i]
		score := r.scoreMatch(query, item.Text, pos, item.Index)

		scores = append(scores, MatchScore{
			Item:          item,
			Score:         score,
			Positions:     pos,
			OriginalIndex: item.Index,
		})
	}

	// Sort by score (descending)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	return scores
}

// scoreMatch calculates a relevance score for a match
func (r *Ranker) scoreMatch(query, text string, positions []int, originalIndex int) float64 {
	if len(positions) == 0 {
		return 0.0
	}

	score := 0.0

	// 1. Compactness: How tightly grouped are the matches?
	// Consecutive matches get maximum score, spread out matches get penalized
	span := positions[len(positions)-1] - positions[0] + 1
	compactness := float64(len(query)) / float64(span)
	score += compactness * r.CompactnessWeight

	// 2. Early match bonus: Matches at the start of text rank higher
	// Position 0 gets full bonus, later positions get diminishing returns
	earlyBonus := 1.0 / float64(positions[0]+1)
	score += earlyBonus * r.EarlyMatchWeight

	// 3. Consecutive match bonus: Reward exact substring matches
	consecutive := 0
	for i := 1; i < len(positions); i++ {
		if positions[i] == positions[i-1]+1 {
			consecutive++
		}
	}
	consecutiveRatio := 0.0
	if len(positions) > 1 {
		consecutiveRatio = float64(consecutive) / float64(len(positions)-1)
	}
	score += consecutiveRatio * r.ConsecutiveWeight

	// 4. Length ratio: Prefer matches where query is significant portion of text
	// "tree" matching "tree" is better than "tree" matching "long/path/to/tree/file.txt"
	lengthRatio := float64(len(query)) / float64(len(text))
	score += lengthRatio * r.LengthRatioWeight

	// 5. Original position weight: Prefer items that appeared earlier in input
	// This acts as a tiebreaker for items with similar match quality
	// Use exponential decay so items far down the list don't dominate
	// Normalize by assuming max 10000 items, so position 0 gets full bonus
	maxItems := 10000.0
	positionScore := 1.0 - (float64(originalIndex) / maxItems)
	if positionScore < 0 {
		positionScore = 0
	}
	score += positionScore * r.OriginalPosWeight

	return score
}
