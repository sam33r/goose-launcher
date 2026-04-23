package main

import (
	"bytes"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	iterations := flag.Int("iterations", 10, "number of launches to time")
	itemCount := flag.Int("items", 100, "items piped to launcher per run")
	keep := flag.Bool("keep-binary", false, "keep ./goose-launcher-bench after run")
	flag.Parse()

	fmt.Println("=== Window Startup Latency Benchmark ===")
	fmt.Printf("Iterations: %d\n", *iterations)
	fmt.Printf("Test items: %d\n\n", *itemCount)

	// Build fresh binary
	fmt.Println("Building goose-launcher...")
	build := exec.Command("go", "build", "-o", "goose-launcher-bench", "./cmd/goose-launcher")
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
		os.Exit(1)
	}

	// Generate test items
	var lines []string
	for i := 1; i <= *itemCount; i++ {
		lines = append(lines, fmt.Sprintf("item_%d", i))
	}
	testData := []byte(strings.Join(lines, "\n") + "\n")

	// Each captured field maps to a column in the BENCHMARK line.
	type runStats struct {
		total, prelaunch, stdinRead, creation, layout, startup []float64
	}
	var stats runStats

	// e.g. "BENCHMARK: total=200.14ms prelaunch=13.61ms stdin=0.01ms creation=50.71ms layout=172.84ms startup=186.49ms items=100"
	metricsRegex := regexp.MustCompile(
		`BENCHMARK: total=([\d.]+)ms prelaunch=([\d.]+)ms stdin=([\d.]+)ms creation=([\d.]+)ms layout=([\d.]+)ms startup=([\d.]+)ms`,
	)

	fmt.Println("Running benchmark...")
	for i := 0; i < *iterations; i++ {
		cmd := exec.Command("./goose-launcher-bench")
		// Stamp launch time *immediately* before exec to capture pre-main work.
		cmd.Env = append(os.Environ(),
			"BENCHMARK_MODE=1",
			fmt.Sprintf("LAUNCH_START_NS=%d", time.Now().UnixNano()),
		)
		cmd.Stdin = bytes.NewReader(testData)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		_ = cmd.Run()

		matches := metricsRegex.FindStringSubmatch(stderr.String())
		if len(matches) != 7 {
			fmt.Printf("Run %2d: failed to parse metrics (stderr=%q)\n", i+1, stderr.String())
			continue
		}
		total := mustFloat(matches[1])
		prelaunch := mustFloat(matches[2])
		stdinRead := mustFloat(matches[3])
		creation := mustFloat(matches[4])
		layout := mustFloat(matches[5])
		startup := mustFloat(matches[6])

		stats.total = append(stats.total, total)
		stats.prelaunch = append(stats.prelaunch, prelaunch)
		stats.stdinRead = append(stats.stdinRead, stdinRead)
		stats.creation = append(stats.creation, creation)
		stats.layout = append(stats.layout, layout)
		stats.startup = append(stats.startup, startup)

		fmt.Printf("Run %2d: total=%.1fms (prelaunch=%.1f stdin=%.1f creation=%.1f layout=%.1f startup=%.1f)\n",
			i+1, total, prelaunch, stdinRead, creation, layout, startup)
	}

	fmt.Println("\n=== Statistics ===")
	printStats("Total (LAUNCH_START_NS -> first frame)", stats.total)
	printStats("Pre-launch (dyld + Go runtime init)", stats.prelaunch)
	printStats("Stdin read", stats.stdinRead)
	printStats("Window creation (theme + font)", stats.creation)
	printStats("Time to first layout", stats.layout)
	printStats("Startup (NewWindow -> first frame)", stats.startup)

	if !*keep {
		_ = os.Remove("./goose-launcher-bench")
	}
}

func mustFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func printStats(name string, values []float64) {
	if len(values) == 0 {
		return
	}
	sort.Float64s(values)
	min := values[0]
	max := values[len(values)-1]

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	median := values[len(values)/2]
	if len(values)%2 == 0 {
		median = (values[len(values)/2-1] + values[len(values)/2]) / 2
	}

	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}
	stddev := math.Sqrt(variance / float64(len(values)))

	fmt.Printf("\n%s:\n", name)
	fmt.Printf("  Min:     %.2f ms\n", min)
	fmt.Printf("  Max:     %.2f ms\n", max)
	fmt.Printf("  Mean:    %.2f ms\n", mean)
	fmt.Printf("  Median:  %.2f ms\n", median)
	fmt.Printf("  Std Dev: %.2f ms\n", stddev)
}
