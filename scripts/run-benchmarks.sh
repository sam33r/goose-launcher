#!/bin/bash
# Performance benchmark runner for goose-launcher

set -e

BENCH_DIR="bench-results"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
RESULT_FILE="${BENCH_DIR}/benchmark-${TIMESTAMP}.txt"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Goose Launcher Performance Benchmarks ===${NC}"
echo "Timestamp: $(date)"
echo "Go version: $(go version)"
echo

# Create results directory
mkdir -p "${BENCH_DIR}"

# Function to run benchmarks for a package
run_bench() {
    local pkg=$1
    local name=$2

    echo -e "${GREEN}Running ${name} benchmarks...${NC}"
    echo "Package: ${pkg}" | tee -a "${RESULT_FILE}"
    echo "----------------------------------------" | tee -a "${RESULT_FILE}"

    go test -bench=. -benchmem -benchtime=1s "${pkg}" | tee -a "${RESULT_FILE}"
    echo | tee -a "${RESULT_FILE}"
}

# Run matcher benchmarks
run_bench "./pkg/matcher" "Matcher (Filtering/Search)"

# Run UI benchmarks
run_bench "./pkg/ui" "UI (Rendering/Layout)"

echo -e "${GREEN}=== Benchmark Summary ===${NC}"
echo "Results saved to: ${RESULT_FILE}"
echo

# Extract key metrics
echo -e "${YELLOW}Key Performance Metrics:${NC}"
echo

echo "Filtering Performance (10k items):"
grep "BenchmarkFuzzyMatch_MediumDataset" "${RESULT_FILE}" || echo "  No data"
grep "BenchmarkWindowFilterItems_Medium" "${RESULT_FILE}" || echo "  No data"
echo

echo "Filtering Performance (100k items):"
grep "BenchmarkFuzzyMatch_LargeDataset" "${RESULT_FILE}" || echo "  No data"
grep "BenchmarkWindowFilterItems_Large" "${RESULT_FILE}" || echo "  No data"
echo

echo "Filtering Performance (1M items):"
grep "BenchmarkFuzzyMatch_VeryLargeDataset" "${RESULT_FILE}" || echo "  No data"
grep "BenchmarkWindowFilterItems_VeryLarge" "${RESULT_FILE}" || echo "  No data"
echo

echo "UI Rendering Performance:"
grep "BenchmarkListLayout" "${RESULT_FILE}" | head -3 || echo "  No data"
echo

echo "Highlighting Overhead:"
grep "BenchmarkListLayout_With.*Highlighting" "${RESULT_FILE}" || echo "  No data"
echo

echo "Progressive Search (typing simulation):"
grep "BenchmarkSearchLatency_Progressive" "${RESULT_FILE}" || echo "  No data"
echo

# Generate comparison if previous benchmarks exist
PREV_FILE=$(ls -t ${BENCH_DIR}/benchmark-*.txt 2>/dev/null | sed -n '2p')
if [ -n "${PREV_FILE}" ]; then
    echo -e "${YELLOW}Comparison with previous run:${NC}"
    echo "Previous: ${PREV_FILE}"
    echo "Current:  ${RESULT_FILE}"
    echo

    if command -v benchcmp &> /dev/null; then
        benchcmp "${PREV_FILE}" "${RESULT_FILE}" || echo "No significant changes"
    else
        echo "Install benchcmp for detailed comparison: go install golang.org/x/tools/cmd/benchcmp@latest"
    fi
fi

echo
echo -e "${GREEN}Done! Full results in ${RESULT_FILE}${NC}"
