#!/bin/bash
# Window startup latency benchmark — thin wrapper around the Go tool that does
# the actual work. The Go tool sets LAUNCH_START_NS so the launcher's
# BENCHMARK_MODE output can attribute pre-main (dyld/runtime) time too.
#
# Usage:
#     scripts/benchmark-startup.sh [iterations] [items]
#
# For an end-to-end wall-clock number that includes process spawn,
# see scripts/hyperfine-launch.sh.

set -e

# Run from the repo root regardless of where this script was invoked.
cd "$(dirname "$0")/.."

ITERATIONS=${1:-10}
ITEMS=${2:-100}

go run ./cmd/benchmark-startup -iterations "$ITERATIONS" -items "$ITEMS"
