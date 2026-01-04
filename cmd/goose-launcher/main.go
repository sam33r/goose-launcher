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

	fmt.Fprintf(os.Stderr, "DEBUG: Loaded %d items\n", len(items))

	// Apply config (exact mode, etc.)
	// TODO: Pass config to window - deferred to future enhancement
	_ = cfg

	// Run UI in goroutine, app.Main() on main thread (required for macOS)
	go func() {
		fmt.Fprintf(os.Stderr, "DEBUG: Creating window...\n")
		window := ui.NewWindow(items)

		fmt.Fprintf(os.Stderr, "DEBUG: Running window...\n")
		selected, err := window.Run()

		fmt.Fprintf(os.Stderr, "DEBUG: Window closed, selected=%q, err=%v\n", selected, err)

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
