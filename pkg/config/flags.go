package config

import (
	"flag"
	"fmt"
	"strings"
)

// Config holds launcher configuration from CLI flags
type Config struct {
	ExactMode        bool
	Rank             bool     // Enable ranking/scoring of matches
	Height           int
	Layout           string
	Keybindings      []string // --bind flags (stored for later parsing)
	HighlightMatches bool     // Highlight matching text in results (default: true)
	Markup           string   // Stdin markup format: "" (off) or "pango"
}

// ParseFlags parses command-line arguments into Config
func ParseFlags(args []string) (*Config, error) {
	cfg := &Config{
		ExactMode:        true,   // Default: exact match mode (changed from false)
		Rank:             false,  // Default: preserve stdin order (no re-sorting)
		Height:           100,    // Default: full height
		Layout:           "default",
		HighlightMatches: true,   // Default: highlight matches enabled
	}

	fs := flag.NewFlagSet("goose-launcher", flag.ContinueOnError)

	// Define flags
	var fuzzy bool
	var noSort bool
	fs.BoolVar(&cfg.ExactMode, "e", true, "exact match mode (default: true)")
	fs.BoolVar(&cfg.ExactMode, "exact", true, "exact match mode (default: true)")
	fs.BoolVar(&fuzzy, "fuzzy", false, "fuzzy match mode (overrides --exact)")
	fs.BoolVar(&cfg.Rank, "rank", false, "rank results by match quality (default: false)")
	fs.BoolVar(&noSort, "no-sort", false, "filter only; preserve input order (default; kept for compatibility)")
	fs.IntVar(&cfg.Height, "height", 100, "window height (percentage)")
	fs.StringVar(&cfg.Layout, "layout", "default", "layout style (default|reverse)")
	fs.BoolVar(&cfg.HighlightMatches, "highlight-matches", true, "highlight matching text in results")
	fs.StringVar(&cfg.Markup, "markup", "", "stdin markup format: pango (default: off)")

	// Custom handling for --bind flags (can appear multiple times)
	var bindFlags multiFlag
	fs.Var(&bindFlags, "bind", "custom key bindings")

	// Parse
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// If --fuzzy is passed, it overrides default ExactMode=true
	if fuzzy {
		cfg.ExactMode = false
	}

	// --no-sort forces ranking off regardless of --rank
	if noSort {
		cfg.Rank = false
	}

	// Reject unknown markup formats early so callers see a clear error.
	switch cfg.Markup {
	case "", "pango":
		// ok
	default:
		return nil, fmt.Errorf("unsupported --markup value %q (want \"\" or \"pango\")", cfg.Markup)
	}

	cfg.Keybindings = bindFlags

	return cfg, nil
}

// multiFlag allows multiple values for a flag
type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}
