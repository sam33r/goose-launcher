package main

import (
	"fmt"
	"os"

	"gioui.org/app"

	"github.com/sam33r/goose-launcher/pkg/config"
	"github.com/sam33r/goose-launcher/pkg/input"
	"github.com/sam33r/goose-launcher/pkg/ui"
)

func main() {
	// Parse CLI flags
	cfg, err := config.ParseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Read items from stdin
	reader := input.NewReader(os.Stdin)
	items, err := reader.ReadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	// Run UI in goroutine, app.Main() on main thread (required for macOS)
	go func() {
		window := ui.NewWindow(items, cfg.HighlightMatches, cfg.ExactMode, cfg.Rank)
		selected, err := window.Run()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running window: %v\n", err)
			os.Exit(1)
		}

		// Output selection
		if selected != "" {
			fmt.Println(selected)
		}

		os.Exit(0)
	}()

	// Required for macOS - runs the main event loop
	app.Main()
}
