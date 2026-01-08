#!/bin/bash
# Test script for goose-launcher
# Rebuilds binaries and runs with test data
#
# Usage: ./test-launcher.sh [item_count] [launcher_flags...]
# Examples:
#   ./test-launcher.sh 100              # 100 items, default flags
#   ./test-launcher.sh 100 --fuzzy      # 100 items, fuzzy mode
#   ./test-launcher.sh 100 --rank=false # 100 items, no ranking
#   ./test-launcher.sh 50 --fuzzy --rank=false  # 50 items, fuzzy, no ranking

set -e  # Exit on error

echo "==> Building generate-dataset..."
go build -o generate-dataset ./cmd/generate-dataset

echo "==> Building goose-launcher..."
go build -o goose-launcher ./cmd/goose-launcher

# Default to 100 items, but allow override
ITEM_COUNT=${1:-100}

# Shift to get remaining args as launcher flags
shift || true  # Don't fail if no args
LAUNCHER_FLAGS="$@"

echo "==> Running launcher with $ITEM_COUNT items..."
if [ -n "$LAUNCHER_FLAGS" ]; then
    echo "==> Launcher flags: $LAUNCHER_FLAGS"
fi
echo ""

./generate-dataset -count "$ITEM_COUNT" | ./goose-launcher $LAUNCHER_FLAGS
