# Window Startup Benchmark

Measures the window initialization and rendering performance of goose-launcher.

## Usage

```bash
# Build and run benchmark
go run ./cmd/benchmark-startup

# Or build first
go build -o benchmark-startup ./cmd/benchmark-startup
./benchmark-startup
```

## Metrics Collected

The benchmark measures three key timing points:

1. **Creation Time**: Time from `NewWindow()` start to completion
   - Includes: theme setup, font parsing, component initialization

2. **Layout Time**: Time from window creation to first layout operation
   - Includes: creation time + time until first `layout()` call

3. **Startup Time**: Time from window creation to first frame rendered
   - Includes: layout time + first frame rendering
   - **This is the total user-perceived latency**

## Baseline Performance (Current)

```
Startup Time:
  Min:     147.60 ms
  Max:     373.29 ms
  Mean:    258.34 ms
  Median:  271.13 ms
  Std Dev: 66.52 ms
  Range:   225.69 ms
```

## How It Works

1. Builds a special benchmark binary with `BENCHMARK_MODE` enabled
2. Runs the launcher 10 times with 100 test items
3. Each run automatically closes after the first frame renders
4. Timing is instrumented in the window code:
   - `metrics.WindowCreationStart` - Start of NewWindow()
   - `metrics.WindowCreationEnd` - End of NewWindow()
   - `metrics.FirstLayoutTime` - First layout() call
   - `metrics.FirstFrameTime` - First frame rendered

5. Statistics are calculated across all runs

## Interpreting Results

- **Mean**: Average latency across all runs
- **Median**: Middle value (less affected by outliers)
- **Std Dev**: Variability - lower is more consistent
- **Min**: Best-case performance
- **Max**: Worst-case performance (may include OS scheduler delays)

## Optimization Tracking

Use this benchmark to validate performance improvements:

```bash
# Before optimization
./benchmark-startup > baseline.txt

# After optimization
./benchmark-startup > optimized.txt

# Compare
diff baseline.txt optimized.txt
```

## Known Factors Affecting Performance

- **Font parsing**: ~20-50ms (happens during creation)
- **Theme initialization**: ~10-20ms
- **macOS window creation**: ~30-80ms (OS-level)
- **First frame rendering**: ~10-30ms
- **OS process scheduling**: Variable (causes high std dev)
