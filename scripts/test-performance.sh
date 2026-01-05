#!/bin/bash
# Manual performance testing script for goose-launcher
# Tests the launcher with various dataset sizes

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}=== Goose Launcher Performance Test ===${NC}"
echo

# Build the launcher and dataset generator
echo -e "${GREEN}Building tools...${NC}"
go build -o goose-launcher ./cmd/goose-launcher
go build -o generate-dataset ./cmd/generate-dataset
echo

# Function to test with dataset
test_dataset() {
    local count=$1
    local desc=$2

    echo -e "${YELLOW}Testing with ${desc} (${count} items)${NC}"

    # Generate dataset
    ./generate-dataset -count "${count}" -type mixed > /tmp/test-dataset.txt

    # Measure startup time
    local start=$(date +%s%N)
    # Run launcher in background and kill it immediately
    # We're just testing startup and initial render time
    timeout 0.5s ./goose-launcher < /tmp/test-dataset.txt > /dev/null 2>&1 || true
    local end=$(date +%s%N)

    local duration=$(( (end - start) / 1000000 )) # Convert to milliseconds
    echo "  Startup time: ${duration}ms"

    # Get file size
    local size=$(wc -c < /tmp/test-dataset.txt)
    local size_kb=$(( size / 1024 ))
    echo "  Dataset size: ${size_kb}KB"
    echo
}

# Test different dataset sizes
test_dataset 100 "Small dataset"
test_dataset 1000 "Medium dataset"
test_dataset 10000 "Large dataset"
test_dataset 100000 "Very large dataset"

echo -e "${GREEN}=== Interactive Test ===${NC}"
echo
echo "Now let's test interactively with 10k items."
echo "Try typing different search queries to feel the latency."
echo
echo "Press Enter to start (Ctrl+C or ESC to exit)..."
read

./generate-dataset -count 10000 -type mixed | ./goose-launcher

echo
echo -e "${GREEN}Test complete!${NC}"
echo
echo "To test with different sizes, run:"
echo "  ./generate-dataset -count 100000 | ./goose-launcher"
echo
echo "To disable highlighting (faster):"
echo "  ./generate-dataset -count 100000 | ./goose-launcher --highlight-matches=false"

# Cleanup
rm -f /tmp/test-dataset.txt
