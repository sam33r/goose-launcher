#!/bin/bash
# End-to-end launch-latency benchmark using hyperfine.
#
# This measures the full wall-clock cost from process spawn to exit, including
# dyld + Go runtime init that the in-process metrics in `cmd/benchmark-startup`
# can't see. The launcher exits automatically after the first frame thanks to
# BENCHMARK_MODE=1, so each run completes deterministically.
#
# For an attributed breakdown (creation, layout, startup, etc.), use
# `go run ./cmd/benchmark-startup` instead.

set -e

# Run from repo root regardless of where this was invoked.
cd "$(dirname "$0")/.."

ITEMS=${1:-100}
RUNS=${RUNS:-30}
WARMUP=${WARMUP:-3}

if ! command -v hyperfine >/dev/null 2>&1; then
    cat <<'EOF' >&2
hyperfine not found. Install with one of:
    brew install hyperfine
    cargo install hyperfine
See https://github.com/sharkdp/hyperfine.
EOF
    exit 1
fi

echo "Building goose-launcher..."
go build -o goose-launcher-bench ./cmd/goose-launcher

echo "Generating ${ITEMS} test items..."
DATA_FILE=$(mktemp)
trap 'rm -f "$DATA_FILE" goose-launcher-bench' EXIT
seq 1 "$ITEMS" | awk '{print "item_" $1}' > "$DATA_FILE"

echo "Running hyperfine (${RUNS} runs, ${WARMUP} warmup)..."
echo

# --input pipes the file to stdin on each run; BENCHMARK_MODE makes the
# binary auto-exit after first frame.
BENCHMARK_MODE=1 hyperfine \
    --warmup "$WARMUP" \
    --runs "$RUNS" \
    --input "$DATA_FILE" \
    --shell=none \
    --time-unit millisecond \
    --command-name "goose-launcher (${ITEMS} items)" \
    './goose-launcher-bench'
