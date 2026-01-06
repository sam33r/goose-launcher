package config

import (
	"testing"
)

func TestParseFlags_Exact(t *testing.T) {
	args := []string{"-e"}
	cfg, err := ParseFlags(args)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.ExactMode {
		t.Error("expected ExactMode to be true with -e flag")
	}
}

func TestParseFlags_NoSort(t *testing.T) {
	args := []string{"--no-sort"}
	cfg, err := ParseFlags(args)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.NoSort {
		t.Error("expected NoSort to be true with --no-sort flag")
	}
}

func TestParseFlags_Height(t *testing.T) {
	args := []string{"--height=80"}
	cfg, err := ParseFlags(args)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Height != 80 {
		t.Errorf("expected Height 80, got %d", cfg.Height)
	}
}

func TestParseFlags_Multiple(t *testing.T) {
	args := []string{"-e", "--no-sort", "--height=100", "--layout=reverse"}
	cfg, err := ParseFlags(args)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.ExactMode {
		t.Error("expected ExactMode true")
	}
	if !cfg.NoSort {
		t.Error("expected NoSort true")
	}
	if cfg.Height != 100 {
		t.Errorf("expected Height 100, got %d", cfg.Height)
	}
	if cfg.Layout != "reverse" {
		t.Errorf("expected Layout 'reverse', got '%s'", cfg.Layout)
	}
}

func TestParseFlags_Defaults(t *testing.T) {
	args := []string{}
	cfg, err := ParseFlags(args)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.ExactMode {
		t.Error("expected ExactMode true by default")
	}
	if !cfg.NoSort {
		t.Error("expected NoSort true by default (fzf compatibility)")
	}
	if cfg.Height != 100 {
		t.Errorf("expected default Height 100, got %d", cfg.Height)
	}
	if !cfg.HighlightMatches {
		t.Error("expected HighlightMatches true by default")
	}
}

func TestParseFlags_Fuzzy(t *testing.T) {
	args := []string{"--fuzzy"}
	cfg, err := ParseFlags(args)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ExactMode {
		t.Error("expected ExactMode to be false with --fuzzy flag")
	}
}

// TestHighlightMatchesDefaultTrue tests that --highlight-matches defaults to true
func TestHighlightMatchesDefaultTrue(t *testing.T) {
	cfg, err := ParseFlags([]string{})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if !cfg.HighlightMatches {
		t.Error("HighlightMatches should default to true")
	}
}

// TestHighlightMatchesCanBeEnabled tests that --highlight-matches=true works
func TestHighlightMatchesCanBeEnabled(t *testing.T) {
	cfg, err := ParseFlags([]string{"--highlight-matches=true"})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if !cfg.HighlightMatches {
		t.Error("HighlightMatches should be true when --highlight-matches=true")
	}
}

// TestHighlightMatchesCanBeDisabled tests that --highlight-matches=false works
func TestHighlightMatchesCanBeDisabled(t *testing.T) {
	cfg, err := ParseFlags([]string{"--highlight-matches=false"})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if cfg.HighlightMatches {
		t.Error("HighlightMatches should be false when --highlight-matches=false")
	}
}

// TestHighlightMatchesWithOtherFlags tests that highlight-matches works with other flags
func TestHighlightMatchesWithOtherFlags(t *testing.T) {
	cfg, err := ParseFlags([]string{
		"--exact",
		"--highlight-matches=false",
		"--height=50",
	})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if cfg.HighlightMatches {
		t.Error("HighlightMatches should be false")
	}

	if !cfg.ExactMode {
		t.Error("ExactMode should be true")
	}

	if cfg.Height != 50 {
		t.Errorf("Height = %d, want 50", cfg.Height)
	}
}
