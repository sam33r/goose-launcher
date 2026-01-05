package config

import (
	"flag"
	"strings"
)

// Config holds launcher configuration from CLI flags
type Config struct {
	ExactMode        bool
	NoSort           bool
	Height           int
	Layout           string
	Keybindings      []string // --bind flags (stored for later parsing)
	Interactive      bool
	HighlightMatches bool // Highlight matching text in results (default: true)
}

// ParseFlags parses command-line arguments into Config
func ParseFlags(args []string) (*Config, error) {
	cfg := &Config{
		NoSort:           true,   // Default: maintain input order (fzf compatibility)
		Height:           100,    // Default: full height
		Layout:           "default",
		HighlightMatches: true,   // Default: highlight matches enabled
	}

	fs := flag.NewFlagSet("goose-launcher", flag.ContinueOnError)

	// Define flags
	fs.BoolVar(&cfg.ExactMode, "e", false, "exact match mode")
	fs.BoolVar(&cfg.ExactMode, "exact", false, "exact match mode")
	fs.BoolVar(&cfg.NoSort, "no-sort", true, "do not sort results")
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
