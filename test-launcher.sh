#!/bin/bash
# Test script for goose-launcher
# Rebuilds binaries and runs with test data
#
# Usage:
#   ./test-launcher.sh [item_count] [launcher_flags...]
#   ./test-launcher.sh --pango [launcher_flags...]           # curated sample
#   ./test-launcher.sh --pango N [launcher_flags...]         # N generated items, each wrapped with markup
#
# Examples:
#   ./test-launcher.sh 100                  # 100 items, default flags
#   ./test-launcher.sh 100 --fuzzy          # 100 items, fuzzy mode
#   ./test-launcher.sh --pango              # curated markup sample
#   ./test-launcher.sh --pango 10000        # 10k markup-wrapped items (perf test)
#   ./test-launcher.sh --pango 50000 --fuzzy

set -e  # Exit on error

echo "==> Building generate-dataset..."
go build -o generate-dataset ./cmd/generate-dataset

echo "==> Building goose-launcher..."
go build -o goose-launcher ./cmd/goose-launcher

MODE="dataset"
if [ "$1" = "--pango" ]; then
    MODE="pango"
    shift
fi

if [ "$MODE" = "pango" ]; then
    # Optional item count as the first arg after --pango
    PANGO_COUNT=""
    if [[ "$1" =~ ^[0-9]+$ ]]; then
        PANGO_COUNT="$1"
        shift
    fi

    LAUNCHER_FLAGS="--markup=pango $@"

    if [ -n "$PANGO_COUNT" ]; then
        echo "==> Running launcher with $PANGO_COUNT markup-wrapped items..."
        echo "==> Launcher flags: $LAUNCHER_FLAGS"
        echo ""
        ./generate-dataset -count "$PANGO_COUNT" -markup pango | ./goose-launcher $LAUNCHER_FLAGS
    else
        echo "==> Running launcher with curated Pango markup sample..."
        echo "==> Launcher flags: $LAUNCHER_FLAGS"
        echo ""
        # Curated sample exercises:
        #   - bold / italic / fg color (rendered)
        #   - named colors, hex colors, short hex
        #   - nested and adjacent tags
        #   - <u> and <span background=…> (parsed, deferred — render as plain)
        #   - entity decoding (&lt; &amp;)
        #   - malformed line (should fall back to literal, no crash)
        #   - plugin-style separator line (selection output should be markup-free)
        cat <<'EOF' | ./goose-launcher $LAUNCHER_FLAGS
<b>ERROR</b>    . connection refused
<span foreground="#4ec9b0">OK</span>       . server ready
<span fg="yellow">WARN</span>     . disk nearly full
<i>apricot</i>
<b>apple</b>
application
<b><i>bold italic</i></b> mixed
<span foreground="#f00">red</span><span foreground="#0f0">green</span><span foreground="#00f">blue</span> adjacent
<span foreground="lightmagenta">light magenta named color</span>
<span foreground="#abc">short hex 3-char</span>
<u>underline is parsed but not yet rendered</u>
<span background="#444">background is parsed but not yet rendered</span>
entities: 2 &lt; 3 &amp;&amp; 3 &gt; 2
<unterminated malformed markup falls back to literal text
plain line with no markup
plugin-x    . <b>styled</b> selection stays clean on stdout
EOF
    fi
else
    ITEM_COUNT=${1:-100}
    shift || true
    LAUNCHER_FLAGS="$@"

    echo "==> Running launcher with $ITEM_COUNT items..."
    if [ -n "$LAUNCHER_FLAGS" ]; then
        echo "==> Launcher flags: $LAUNCHER_FLAGS"
    fi
    echo ""

    ./generate-dataset -count "$ITEM_COUNT" | ./goose-launcher $LAUNCHER_FLAGS
fi
