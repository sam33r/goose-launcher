package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gioui.org/app"

	"github.com/sam33r/goose-launcher/pkg/config"
	"github.com/sam33r/goose-launcher/pkg/input"
	"github.com/sam33r/goose-launcher/pkg/ui"
)

// procStart is captured during package initialization — the earliest moment
// in user code, after the Go runtime has finished setting up. Subtracting
// this from a parent-supplied LAUNCH_START_NS gives the dyld + runtime-init
// portion of cold-start latency.
var procStart = time.Now()

func main() {
	// Check if benchmark mode is enabled
	if os.Getenv("BENCHMARK_MODE") == "1" {
		ui.BenchmarkMode = true
	}

	// Optional: parent harness can record exec-time in nanoseconds-since-epoch
	// (matching time.Now().UnixNano()) so we can attribute the dyld + runtime
	// chunk that's invisible to in-process timers.
	var launchStart time.Time
	if v := os.Getenv("LAUNCH_START_NS"); v != "" {
		if ns, err := strconv.ParseInt(v, 10, 64); err == nil {
			launchStart = time.Unix(0, ns)
		}
	}

	// Parse CLI flags
	cfg, err := config.ParseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Read items from stdin
	stdinStart := time.Now()
	reader := input.NewReader(os.Stdin, cfg.Markup)
	items, err := reader.ReadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}
	stdinEnd := time.Now()

	// Run UI in goroutine, app.Main() on main thread (required for macOS)
	go func() {
		window := ui.NewWindow(items, cfg.HighlightMatches, cfg.ExactMode, cfg.Rank)
		if ui.BenchmarkMode {
			window.SetEarlyMetrics(launchStart, procStart, stdinStart, stdinEnd)
		}
		selected, err := window.Run()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running window: %v\n", err)
			os.Exit(1)
		}

		// Output benchmark metrics if enabled
		if ui.BenchmarkMode {
			metrics := window.GetMetrics()
			fmt.Fprintf(os.Stderr,
				"BENCHMARK: total=%.2fms prelaunch=%.2fms stdin=%.2fms creation=%.2fms layout=%.2fms startup=%.2fms items=%d\n",
				ms(metrics.GetTotalDuration()),
				ms(metrics.GetPrelaunchDuration()),
				ms(metrics.GetStdinReadDuration()),
				ms(metrics.GetCreationDuration()),
				ms(metrics.GetTimeToFirstLayout()),
				ms(metrics.GetStartupDuration()),
				len(items),
			)
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

func ms(d time.Duration) float64 { return d.Seconds() * 1000 }
