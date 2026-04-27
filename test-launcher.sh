#!/bin/bash
# Test script for goose-launcher
# Rebuilds binaries and runs with test data
#
# Usage:
#   ./test-launcher.sh [item_count] [launcher_flags...]
#   ./test-launcher.sh --pango [launcher_flags...]              # curated sample
#   ./test-launcher.sh --pango N [launcher_flags...]            # N generated items, each wrapped with markup
#   ./test-launcher.sh --stream [COUNT] [DELAY_MS] [flags...]   # slow producer; exercises streaming stdin
#
# --stream mode: the launcher window must appear within ~200 ms even though
# the shell loop takes COUNT * DELAY_MS ms to finish. The "X/Y" count climbs
# from 0 toward COUNT as items arrive. Press Enter early to confirm the
# producer terminates promptly (the daemon closes the socket; the shell loop
# gets SIGPIPE on its next echo).
#
# Examples:
#   ./test-launcher.sh 100                  # 100 items, default flags
#   ./test-launcher.sh 100 --fuzzy          # 100 items, fuzzy mode
#   ./test-launcher.sh --pango              # curated markup sample
#   ./test-launcher.sh --pango 10000        # 10k markup-wrapped items (perf test)
#   ./test-launcher.sh --pango 50000 --fuzzy
#   ./test-launcher.sh --stream             # 50 items, 100 ms apart (default)
#   ./test-launcher.sh --stream 200 50      # 200 items, 50 ms apart
#   ./test-launcher.sh --stream 30 200 --rank

set -e  # Exit on error

echo "==> Building generate-dataset..."
go build -o generate-dataset ./cmd/generate-dataset

echo "==> Building goose-launcher..."
go build -o goose-launcher ./cmd/goose-launcher

echo "==> Building goose-launcherd..."
go build -o goose-launcherd ./cmd/goose-launcherd

MODE="dataset"
if [ "$1" = "--pango" ]; then
    MODE="pango"
    shift
elif [ "$1" = "--stream" ]; then
    MODE="stream"
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
elif [ "$MODE" = "stream" ]; then
    STREAM_COUNT=50
    STREAM_DELAY_MS=100
    if [[ "$1" =~ ^[0-9]+$ ]]; then
        STREAM_COUNT="$1"
        shift
    fi
    if [[ "$1" =~ ^[0-9]+$ ]]; then
        STREAM_DELAY_MS="$1"
        shift
    fi
    LAUNCHER_FLAGS="$@"

    DELAY_S=$(awk "BEGIN { printf \"%.3f\", $STREAM_DELAY_MS / 1000 }")
    TOTAL_S=$(awk "BEGIN { printf \"%.1f\", ($STREAM_COUNT * $STREAM_DELAY_MS) / 1000 }")

    echo "==> Streaming $STREAM_COUNT items, one every ${STREAM_DELAY_MS} ms (~${TOTAL_S}s total)"
    echo "    Window should appear immediately; X/Y count climbs from 0/0 to $STREAM_COUNT/$STREAM_COUNT."
    echo "    Press Enter early to verify cancellation kills the producer."
    if [ -n "$LAUNCHER_FLAGS" ]; then
        echo "==> Launcher flags: $LAUNCHER_FLAGS"
    fi
    echo ""

    time ( for i in $(seq 1 "$STREAM_COUNT"); do
        echo "stream-item-$i"
        sleep "$DELAY_S"
    done ) | ./goose-launcher $LAUNCHER_FLAGS
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
