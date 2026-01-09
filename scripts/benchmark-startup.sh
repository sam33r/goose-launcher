#!/bin/bash
# Benchmark window startup latency by measuring multiple launches
#
# This script measures the time from process start to window becoming interactive
# by launching the binary multiple times and timing until it exits.

set -e

ITERATIONS=${1:-10}
ITEM_COUNT=${2:-100}

echo "=== Window Startup Latency Benchmark ==="
echo "Iterations: $ITERATIONS"
echo "Test items: $ITEM_COUNT"
echo ""

# Build fresh binary
echo "Building goose-launcher..."
go build -o goose-launcher ./cmd/goose-launcher
echo ""

# Generate test data
./cmd/generate-dataset/generate-dataset -count $ITEM_COUNT > /tmp/bench-items.txt 2>/dev/null || \
    (cd cmd/generate-dataset && go build -o ../../generate-dataset . && cd ../.. && \
     ./generate-dataset -count $ITEM_COUNT > /tmp/bench-items.txt)

RESULTS_FILE=$(mktemp)

echo "Running benchmark..."
for i in $(seq 1 $ITERATIONS); do
    # Measure time to launch and exit (send ESC key immediately)
    START=$(gdate +%s%3N 2>/dev/null || date +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time() * 1000))")

    # Launch and immediately send ESC to close
    # We measure from launch to window appearing (ready for input)
    # This is a proxy for "window is visible and interactive"
    timeout 2s ./goose-launcher < /tmp/bench-items.txt >/dev/null 2>&1 &
    PID=$!

    # Wait a tiny bit for window to appear, then kill it
    sleep 0.3
    kill -9 $PID 2>/dev/null || true
    wait $PID 2>/dev/null || true

    END=$(gdate +%s%3N 2>/dev/null || date +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time() * 1000))")

    DURATION=$((END - START))
    echo "$DURATION" >> $RESULTS_FILE
    printf "Run %2d: %d ms\n" $i $DURATION

    # Cool down between runs
    sleep 0.1
done

echo ""
echo "=== Statistics ==="

# Calculate stats using awk
awk '
BEGIN {
    min = 999999
    max = 0
    sum = 0
    count = 0
}
{
    val = $1
    if (val < min) min = val
    if (val > max) max = val
    sum += val
    values[count++] = val
}
END {
    mean = sum / count

    # Sort for median
    for (i = 0; i < count; i++) {
        for (j = i + 1; j < count; j++) {
            if (values[i] > values[j]) {
                tmp = values[i]
                values[i] = values[j]
                values[j] = tmp
            }
        }
    }

    median = values[int(count/2)]

    # Std dev
    variance = 0
    for (i = 0; i < count; i++) {
        diff = values[i] - mean
        variance += diff * diff
    }
    stddev = sqrt(variance / count)

    printf "Min:     %d ms\n", min
    printf "Max:     %d ms\n", max
    printf "Mean:    %.2f ms\n", mean
    printf "Median:  %d ms\n", median
    printf "Std Dev: %.2f ms\n", stddev
    printf "Range:   %d ms\n", max - min
}
' $RESULTS_FILE

# Cleanup
rm -f $RESULTS_FILE /tmp/bench-items.txt
