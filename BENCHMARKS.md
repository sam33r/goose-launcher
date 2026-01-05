# Performance Benchmarks

This document describes the performance characteristics of goose-launcher and how to run benchmarks.

## Quick Start

```bash
# Run all benchmarks
./scripts/run-benchmarks.sh

# Interactive performance test
./scripts/test-performance.sh

# Test with specific dataset size
./generate-dataset -count 100000 | ./goose-launcher
```

## Benchmark Results

Results from Apple M2 Pro (tested on darwin/arm64):

### Filtering Performance (Matcher)

| Dataset Size | Time/op | Memory/op | Allocs/op |
|--------------|---------|-----------|-----------|
| 100 items    | ~22µs   | 9 KB      | 118       |
| 10k items    | ~2.4ms  | 1.4 MB    | 15.6k     |
| 100k items   | ~25ms   | 15 MB     | 160k      |
| 1M items     | ~251ms  | 153 MB    | 1.6M      |

**Key Insights:**
- Linear scaling with dataset size
- ~0.25µs per item on average
- Position tracking adds minimal overhead (~1% slower than boolean match)
- Exact mode is ~2x faster than fuzzy mode
- Short queries (1-2 chars) are ~20% faster than long queries

### Window Filtering Performance (UI + Matcher)

| Dataset Size | Time/op | Memory/op | Allocs/op |
|--------------|---------|-----------|-----------|
| 100 items    | ~22µs   | 12 KB     | 128       |
| 10k items    | ~2.5ms  | 1.7 MB    | 15.6k     |
| 100k items   | ~25ms   | 19 MB     | 160k      |
| 1M items     | ~275ms  | 196 MB    | 1.6M      |

**Key Insights:**
- Adds ~10% overhead over raw matcher (due to matchPositions map management)
- Memory overhead is ~30% for position tracking
- Still maintains linear scaling

### UI Rendering Performance (List Layout)

| Test Scenario | Time/op | Memory/op | Allocs/op |
|---------------|---------|-----------|-----------|
| 100 items (no highlight)    | ~51µs   | 48 KB     | 0         |
| 1000 items (no highlight)   | ~47µs   | 54 KB     | 0         |
| 10k items (no highlight)    | ~48µs   | 53 KB     | 0         |
| 1000 items (with highlight) | ~110µs  | 115 KB    | 1046      |

**Key Insights:**
- **Rendering time is constant regardless of dataset size!** (due to virtualization)
- Highlighting adds ~2.3x overhead to rendering (110µs vs 47µs)
- Highlighting overhead is worth the UX benefit for most use cases
- No allocations for non-highlighted rendering (very efficient)

### Text Highlighting Performance

| Operation | Time/op | Memory/op | Allocs/op |
|-----------|---------|-----------|-----------|
| Single highlighted text | ~7.6µs  | 8.6 KB    | 70        |

**Key Insights:**
- Highlighting a single item takes ~7.6µs
- For 1000 items with highlighting: 1000 × 7.6µs = 7.6ms total
- Still well within interactive latency budgets (<16ms for 60fps)

### Progressive Search (Typing Simulation)

Progressive search test simulates typing "handler" character by character:
`h → ha → han → hand → handl → handle → handler`

| Dataset Size | Time/op (7 searches) |
|--------------|---------------------|
| 100k items   | ~175ms              |

**Key Insights:**
- ~25ms per character typed on 100k dataset
- Stays below 60fps threshold (16ms) for datasets up to ~60k items
- For larger datasets, consider debouncing or incremental filtering

## Performance Recommendations

### For Different Dataset Sizes

**< 10k items:**
- All features enabled, instant responsiveness
- Highlighting recommended for better UX
- Expected latency: < 3ms per keystroke

**10k - 100k items:**
- Highlighting still viable
- Consider debouncing search input (50-100ms)
- Expected latency: 3-30ms per keystroke

**100k - 1M items:**
- Consider disabling highlighting: `--highlight-matches=false`
- Implement debouncing (100-200ms)
- Expected latency: 30-300ms per keystroke
- Filtering is still functional but may feel sluggish

**> 1M items:**
- Disable highlighting
- Consider implementing incremental search or index-based filtering
- May need architectural changes for sub-100ms latency

### Memory Considerations

Memory usage is approximately:
- **Base:** 20 bytes per item (string storage)
- **Filtering:** +15 bytes per item (position arrays)
- **Total:** ~35 bytes per item when filtering

For 1M items: ~35 MB memory usage during active filtering.

## Running Benchmarks

### Built-in Go Benchmarks

```bash
# Run matcher benchmarks
go test -bench=. -benchmem ./pkg/matcher

# Run UI benchmarks
go test -bench=. -benchmem ./pkg/ui

# Run specific benchmark
go test -bench=BenchmarkFuzzyMatch_LargeDataset -benchmem ./pkg/matcher

# Run with longer benchmark time for more accurate results
go test -bench=. -benchmem -benchtime=5s ./pkg/matcher

# Save results for comparison
go test -bench=. -benchmem ./pkg/matcher > bench-old.txt
# ... make changes ...
go test -bench=. -benchmem ./pkg/matcher > bench-new.txt
benchcmp bench-old.txt bench-new.txt
```

### Automated Benchmark Script

```bash
# Run comprehensive benchmarks with summary
./scripts/run-benchmarks.sh

# Results saved to: bench-results/benchmark-TIMESTAMP.txt
```

### Interactive Performance Testing

```bash
# Build and test with increasing dataset sizes
./scripts/test-performance.sh

# This will:
# 1. Test startup time with 100, 1k, 10k, 100k items
# 2. Launch interactive test with 10k items
# 3. Report timings
```

### Dataset Generator

Generate test datasets for manual testing:

```bash
# Build the generator
go build -o generate-dataset ./cmd/generate-dataset

# Generate 10k file paths
./generate-dataset -count 10000 -type paths > test-data.txt

# Generate 100k commands
./generate-dataset -count 100000 -type commands > test-data.txt

# Generate mixed data (realistic)
./generate-dataset -count 50000 -type mixed > test-data.txt

# Use with launcher
./generate-dataset -count 10000 | ./goose-launcher

# Test with highlighting disabled
./generate-dataset -count 100000 | ./goose-launcher --highlight-matches=false

# Use reproducible random seed
./generate-dataset -count 10000 -seed 12345 -type mixed > test-data.txt
```

### Data Types

The dataset generator supports three types:

1. **paths** - Realistic file paths (e.g., `src/service/handler_123.go`)
2. **commands** - Command-line commands (e.g., `git commit -m production_config_42`)
3. **mixed** - Mix of paths (50%), commands (30%), and other (20%)

## Performance Monitoring

### Profiling

```bash
# CPU profiling
go test -bench=BenchmarkFuzzyMatch_LargeDataset -cpuprofile=cpu.prof ./pkg/matcher
go tool pprof cpu.prof

# Memory profiling
go test -bench=BenchmarkFuzzyMatch_LargeDataset -memprofile=mem.prof ./pkg/matcher
go tool pprof mem.prof

# Generate profile visualization
go tool pprof -http=:8080 cpu.prof
```

### Real-world Testing

```bash
# Test with actual file listing (macOS)
find . -type f | ./goose-launcher

# Test with git log
git log --oneline | ./goose-launcher

# Test with process list
ps aux | ./goose-launcher

# Test with large command history
history | ./goose-launcher
```

## Optimization Tips

### For Maintainers

1. **Filtering is the bottleneck** - Optimize matcher.Match() for best results
2. **Rendering is O(1)** - Gio's virtualization handles large lists efficiently
3. **Highlighting is expensive** - Consider lazy evaluation or caching
4. **Memory allocations** - Position arrays dominate allocation counts

### Potential Optimizations

1. **Incremental filtering**: Only re-filter items that might change results
2. **Position caching**: Cache match positions for unchanged items
3. **Parallel filtering**: Use goroutines for large datasets (>100k items)
4. **String pooling**: Reduce string allocations during matching
5. **Early termination**: Stop after N matches for large datasets

## Regression Testing

Always run benchmarks before/after major changes:

```bash
# Before changes
go test -bench=. -benchmem ./... > benchmarks-before.txt

# After changes
go test -bench=. -benchmem ./... > benchmarks-after.txt

# Compare
benchcmp benchmarks-before.txt benchmarks-after.txt
```

Look for:
- **Time regressions** > 10% slower
- **Memory regressions** > 20% more allocations
- **Allocation count increases** (especially in hot paths)

## Continuous Integration

Add to CI pipeline:

```yaml
# Example GitHub Actions
- name: Run benchmarks
  run: |
    go test -bench=. -benchmem ./... | tee benchmark-results.txt

- name: Check for regressions
  run: |
    # Fail if benchmarks regress significantly
    ./scripts/check-benchmark-regression.sh
```
