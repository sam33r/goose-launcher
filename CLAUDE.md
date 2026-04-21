# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Native macOS launcher binary — a drop-in fzf replacement used by [Goose](https://github.com/sam33r/goosey). Reads items from stdin, shows a Gio-based window, prints the selection to stdout. Exit code 1 on cancel (ESC).

## Commands

```bash
make build          # Build for host arch into ./goose-launcher
make build-macos    # Universal (arm64 + amd64) via lipo
make install        # build-macos then cp to /usr/local/bin
make test           # go test -v ./...

# Single package / test
go test -v ./pkg/matcher
go test -v ./pkg/ui -run TestHighlight

# Benchmarks (Go bench)
go test -bench=. ./pkg/matcher
go test -bench=. ./pkg/ui

# Scripts
./scripts/run-benchmarks.sh         # automated bench suite
./scripts/test-performance.sh       # interactive perf test
./scripts/benchmark-startup.sh      # startup-time harness (uses BENCHMARK_MODE=1)
./test-launcher.sh [count] [flags]  # rebuild + run with generated dataset
./generate-dataset -count N         # writes N synthetic items to stdout
```

`BENCHMARK_MODE=1` makes `main.go` emit `BENCHMARK: startup=… creation=… layout=…` to stderr.

## Architecture

Single Go module (`github.com/sam33r/goose-launcher`, Go 1.25). The main entry point at `cmd/goose-launcher/main.go` is deliberately thin — it parses flags, reads stdin, then runs the UI on a goroutine while `app.Main()` holds the main thread (required by Gio on macOS).

Pipeline:

```
stdin → input.Reader → []input.Item
                         ↓
                    ui.Window.Run()   ← matcher.FuzzyMatcher + ranker.Ranker
                         ↓
                    stdout (selected line)
```

Packages:

- `pkg/input` — stdin reader; parses plugin-style separator lines (see `test-integration.sh` for the format).
- `pkg/matcher` — fuzzy + exact matching. Returns match positions for highlighting. Position tracking adds ~1% overhead vs. boolean match; exact mode is ~2× faster than fuzzy.
- `pkg/ranker` — scores matches so results can be sorted by quality (toggle with `--rank`).
- `pkg/config` — flag parsing. Defaults: `ExactMode=true`, `Rank=false` (preserve stdin order), `HighlightMatches=true`. `--fuzzy` overrides `--exact`; `--no-sort` is the documented alias for the default `Rank=false` behavior. `--bind` is repeatable.
- `pkg/ui` — Gio window, search input, list widget, match highlighting. Fonts are JetBrains Mono TTFs embedded via `//go:embed` and served through `pkg/fontcache`. `list.go` composes Pango span styling with match-highlight color in a single per-rune pass.
- `pkg/markup` — parses a Pango-markup subset (bold, italic, foreground color; `<u>` and `<span background=…>` parsed but not yet rendered) when `--markup=pango` is set. Stripped text is used for matching; selection output never includes markup.
- `pkg/fontcache` — on-disk cache so font parsing doesn't dominate startup.
- `cmd/generate-dataset` — synthetic data generator for tests/benchmarks.
- `cmd/benchmark-startup` — standalone tool that launches the binary repeatedly and aggregates `BENCHMARK_MODE` timings.

### UI threading (Gio / macOS)

macOS requires windowing on the main OS thread. `main.go` follows the required pattern: UI work runs in `go func() { window.Run(); os.Exit(0) }()` and the main goroutine calls `app.Main()`. Do not refactor this to run `window.Run()` on the main goroutine — it will break on macOS. See `TEST-DAEMON.md` for prior investigation into why a persistent daemon variant hung.

### Interaction model

Keybindings match fzf defaults plus customizable `--bind` flags. `Shift+Enter` (and an empty-filter `Enter`) outputs the typed query rather than a selection — integrations rely on this. `docs/USAGE.md` has the canonical `LAUNCHER_CMD` Goose users copy into `~/.config/goose`.

## Performance expectations

Linear-scaling matcher (~0.25µs/item); rendering is O(1) in list size because only visible rows are laid out. Targets: <100ms launch, <50ms filter latency, <16ms frame. For >100k items, callers should pass `--highlight-matches=false`. Full numbers in `BENCHMARKS.md`.

## Conventions

- Never commit built binaries — `.gitignore` covers `goose-launcher`, `goose-launcher-*`, `generate-dataset`, `benchmark-startup`, `test-*`. Stray checked-in binaries at the repo root (e.g. `test-gio`) are legacy; don't add more.
- UI changes that affect layout or keybindings should be verified interactively via `./test-launcher.sh` — Go tests don't render real windows.
