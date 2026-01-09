package main

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
)

const (
	iterations = 10
	itemCount  = 100
)

func main() {
	fmt.Println("=== Window Startup Latency Benchmark ===")
	fmt.Printf("Iterations: %d\n", iterations)
	fmt.Printf("Test items: %d\n\n", itemCount)

	// Build fresh binary
	fmt.Println("Building goose-launcher with benchmark mode...")
	build := exec.Command("go", "build", "-tags", "benchmark", "-o", "goose-launcher-bench", "./cmd/goose-launcher")
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
		os.Exit(1)
	}

	// Generate test items
	generateCmd := exec.Command("sh", "-c", "seq 1 "+fmt.Sprintf("%d", itemCount)+" | awk '{print \"item_\" $1}'")
	testData, err := generateCmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate test data: %v\n", err)
		os.Exit(1)
	}

	results := make([]float64, 0, iterations)
	creationTimes := make([]float64, 0, iterations)
	layoutTimes := make([]float64, 0, iterations)

	metricsRegex := regexp.MustCompile(`BENCHMARK: startup=(\d+\.?\d*)ms creation=(\d+\.?\d*)ms layout=(\d+\.?\d*)ms`)

	fmt.Println("Running benchmark...")
	for i := 0; i < iterations; i++ {
		// Run launcher with benchmark mode enabled
		cmd := exec.Command("./goose-launcher-bench")
		cmd.Env = append(os.Environ(), "BENCHMARK_MODE=1")
		cmd.Stdin = bytes.NewReader(testData)

		output, _ := cmd.CombinedOutput()

		// Parse metrics from output
		matches := metricsRegex.FindStringSubmatch(string(output))
		if len(matches) == 4 {
			startup, _ := strconv.ParseFloat(matches[1], 64)
			creation, _ := strconv.ParseFloat(matches[2], 64)
			layout, _ := strconv.ParseFloat(matches[3], 64)

			results = append(results, startup)
			creationTimes = append(creationTimes, creation)
			layoutTimes = append(layoutTimes, layout)

			fmt.Printf("Run %2d: %.2f ms (creation: %.2f ms, layout: %.2f ms)\n",
				i+1, startup, creation, layout)
		} else {
			fmt.Printf("Run %2d: Failed to parse metrics\n", i+1)
		}
	}

	// Print statistics
	fmt.Println("\n=== Statistics ===")
	printStats("Startup Time", results)
	printStats("Creation Time", creationTimes)
	printStats("Layout Time", layoutTimes)

	// Cleanup
	os.Remove("./goose-launcher-bench")
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
	fmt.Printf("  Range:   %.2f ms\n", max-min)
}
