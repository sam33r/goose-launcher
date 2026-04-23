#!/bin/bash
# Build + install the goose-launcher daemon prototype from source.
#
# Installs both binaries via `go install` (lands in $GOBIN, defaulting to
# $GOPATH/bin) and restarts any running daemon so the new binary takes
# effect on the next invocation.
#
# Usage: ./install.sh

set -e

# Run from the repo root regardless of where this script was invoked.
cd "$(dirname "$0")"

if ! command -v go &>/dev/null; then
    echo "Error: Go is not installed. Get it from https://go.dev/dl/" >&2
    exit 1
fi

GOBIN="$(go env GOBIN)"
if [[ -z "$GOBIN" ]]; then
    GOBIN="$(go env GOPATH)/bin"
fi

echo "=== Goose Launcher Install ==="
echo "Go:     $(go version | awk '{print $3}')"
echo "Target: $GOBIN"
echo

# Stop any running daemon BEFORE installing — macOS will refuse to overwrite
# a binary held open by a running process. The next client invocation
# autostarts a fresh daemon off the new binary.
if pgrep -f "$GOBIN/goose-launcherd" >/dev/null 2>&1; then
    echo "Stopping running daemon..."
    pkill -f "$GOBIN/goose-launcherd" || true
    # Wait up to 1 s for the daemon to actually exit; otherwise the install
    # below races with it.
    for _ in 1 2 3 4 5 6 7 8 9 10; do
        pgrep -f "$GOBIN/goose-launcherd" >/dev/null 2>&1 || break
        sleep 0.1
    done
fi

echo "Building + installing client + daemon..."
go install ./cmd/goose-launcher ./cmd/goose-launcherd
echo "  -> $GOBIN/goose-launcher"
echo "  -> $GOBIN/goose-launcherd"
echo

# Sanity check: client + daemon both reachable on PATH.
if ! command -v goose-launcher &>/dev/null; then
    echo "Warning: $GOBIN is not on your PATH; add it to use 'goose-launcher' bare." >&2
fi

echo "Done."
echo
echo "Try it:    echo -e 'alpha\\nbeta\\ngamma' | goose-launcher"
echo "Stop it:   pkill goose-launcherd"
echo "Architecture: docs/DAEMON-RESEARCH.md"
