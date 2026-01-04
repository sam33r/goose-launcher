package main

import (
	"fmt"
	"os"

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

	// Create and run window
	window := ui.NewWindow(items)

	// Apply config (exact mode, etc.)
	// TODO: Pass config to window - deferred to future enhancement
	_ = cfg

	selected, err := window.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running window: %v\n", err)
		os.Exit(1)
	}

	// Output selection
	if selected != "" {
		fmt.Println(selected)
	} else {
		// No selection (ESC pressed)
		os.Exit(1)
	}
}
