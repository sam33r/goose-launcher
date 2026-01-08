package config

import (
	"flag"
	"strings"
)

// Config holds launcher configuration from CLI flags
type Config struct {
	ExactMode        bool
	Rank             bool     // Enable ranking/scoring of matches
	Height           int
	Layout           string
	Keybindings      []string // --bind flags (stored for later parsing)
	Interactive      bool
	HighlightMatches bool // Highlight matching text in results (default: true)
}

// ParseFlags parses command-line arguments into Config
func ParseFlags(args []string) (*Config, error) {
	cfg := &Config{
		ExactMode:        true,   // Default: exact match mode (changed from false)
		Rank:             true,   // Default: enable ranking
		Height:           100,    // Default: full height
		Layout:           "default",
		HighlightMatches: true,   // Default: highlight matches enabled
	}

	fs := flag.NewFlagSet("goose-launcher", flag.ContinueOnError)

	// Define flags
	var fuzzy bool
	fs.BoolVar(&cfg.ExactMode, "e", true, "exact match mode (default: true)")
	fs.BoolVar(&cfg.ExactMode, "exact", true, "exact match mode (default: true)")
	fs.BoolVar(&fuzzy, "fuzzy", false, "fuzzy match mode (overrides --exact)")
	fs.BoolVar(&cfg.Rank, "rank", true, "rank results by match quality (default: true)")
	fs.IntVar(&cfg.Height, "height", 100, "window height (percentage)")
	fs.StringVar(&cfg.Layout, "layout", "default", "layout style (default|reverse)")
	fs.BoolVar(&cfg.Interactive, "interactive", false, "interactive mode (read stdin continuously)")
	fs.BoolVar(&cfg.HighlightMatches, "highlight-matches", true, "highlight matching text in results")

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
