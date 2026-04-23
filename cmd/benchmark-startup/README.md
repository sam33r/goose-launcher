# Window Startup Benchmark

Measures the launch-to-first-frame latency of goose-launcher with a six-stage
breakdown.

## Usage

```bash
# From the repo root:
go run ./cmd/benchmark-startup                    # defaults: 10 runs, 100 items
go run ./cmd/benchmark-startup -iterations 30 -items 1000

# Or via the wrapper script (works from anywhere):
scripts/benchmark-startup.sh 10 100

# For a single end-to-end wall-clock number using hyperfine:
scripts/hyperfine-launch.sh 100
```

## What it measures

The benchmark stamps `LAUNCH_START_NS` just before `exec()` and the launcher
emits a `BENCHMARK:` line on stderr after the first frame. The runner parses
that line and aggregates min/mean/median/stddev across runs.

| Stage             | What it covers                                          |
| ----------------- | ------------------------------------------------------- |
| `prelaunch`       | dyld + Go runtime init (`exec` -> first user code)      |
| `stdin`           | reading + parsing items from stdin                      |
| `creation`        | `NewWindow()` — theme + JetBrains Mono font setup       |
| `layout`          | `NewWindow` start -> first `layout()` call (Gio cold path) |
| `startup`         | `NewWindow` start -> first frame submitted              |
| `total`           | `LAUNCH_START_NS` -> first frame (user-perceived)       |

`prelaunch` is the only number this benchmark *can* measure that the launcher
itself can't, because the launcher only starts running code after dyld + Go
runtime init are done.

## How it works

1. Builds `./goose-launcher-bench` from `./cmd/goose-launcher`.
2. For each iteration: stamps `time.Now().UnixNano()` into `LAUNCH_START_NS`,
   spawns the binary with `BENCHMARK_MODE=1`, pipes test items to stdin.
3. The binary auto-cancels after the first frame (see
   `pkg/ui/window.go` Run loop) and prints the `BENCHMARK:` line to stderr.
4. Runner parses the line, aggregates per stage, prints stats.

## Caveats

- `prelaunch` has high variance on the first run — dyld cache is cold.
  Subsequent runs settle to ~30 ms on Apple Silicon. Look at median, not mean.
- Window creation is dominated by font face parsing (~50 ms steady). The
  on-disk fontcache (`pkg/fontcache`) helps here but parsing still happens
  per launch.
- macOS WindowServer composition latency (frame -> pixels-on-screen) is *not*
  captured. Use Instruments' "Time Profiler" template if that delta matters.
