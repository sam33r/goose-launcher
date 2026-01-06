# Goose Native Launcher

Native macOS launcher for Goose - a drop-in replacement for fzf with rich UI and better performance.

## Features

✨ **Native macOS UI** - Spotlight-inspired centered window
🚀 **High Performance** - <100ms launch, <50ms filter latency
🎨 **Rich Text Formatting** - Syntax highlighting and visual feedback
⌨️ **Full Keyboard Control** - Compatible with fzf keybindings
🔌 **Drop-in Replacement** - Works with existing Goose plugins

## Status

**MVP Complete** - Ready for testing

See [NATIVE_LAUNCHER_SPEC.md](../goose/mayor/rig/NATIVE_LAUNCHER_SPEC.md) for full specification.

## Installation

### Using `go install` (Recommended)

```bash
go install github.com/sam33r/goose-launcher/cmd/goose-launcher@latest
```

### From Source

```bash
git clone https://github.com/sam33r/goose-launcher.git
cd goose-launcher
make install
```

See [INSTALL.md](INSTALL.md) for detailed installation instructions.

## Quick Start

```bash
# Test
echo -e "Item 1\nItem 2\nItem 3" | goose-launcher

# With files
find . -type f | goose-launcher

# Configure Goose
echo 'LAUNCHER_CMD="goose-launcher --no-sort --height=100"' >> ~/.config/goose
```

## Documentation

- [Installation Guide](INSTALL.md) - Detailed installation methods
- [Usage Guide](docs/USAGE.md) - Command-line usage and features
- [Benchmarks](BENCHMARKS.md) - Performance analysis and profiling
- [Technical Spec](../goose/mayor/rig/NATIVE_LAUNCHER_SPEC.md)
- [Implementation Plan](docs/plans/2026-01-03-native-launcher-mvp.md)

## Architecture

**Tech Stack:**
- Go 1.21+
- [Gio](https://gioui.org) v0.9.0 - Pure Go GUI toolkit
- fzf matching algorithm (ported)

**Project Structure:**
```
cmd/goose-launcher/    # Main executable
pkg/
  ├── input/           # Stdin parsing
  ├── matcher/         # Fuzzy matching
  ├── ui/              # Gio-based UI
  └── config/          # CLI flag parsing
```

## Performance

### Current Benchmarks (Apple M2 Pro)

| Dataset Size | Filter Latency | Memory Usage |
|--------------|----------------|--------------|
| 100 items    | ~22µs          | ~12 KB       |
| 10k items    | ~2.5ms         | ~1.7 MB      |
| 100k items   | ~25ms          | ~19 MB       |
| 1M items     | ~275ms         | ~196 MB      |

**Key Metrics:**
- Launch time: ~50ms ✓
- Rendering: constant O(1) regardless of dataset size ✓
- Highlighting overhead: ~2.3x (still <16ms for 60fps) ✓
- Binary size: 8.3MB (universal) ✓

**Recommendations:**
- **< 10k items**: All features enabled, instant responsiveness
- **10k-100k items**: Consider debouncing (50-100ms)
- **> 100k items**: Disable highlighting with `--highlight-matches=false`

See [BENCHMARKS.md](BENCHMARKS.md) for detailed performance analysis and profiling guides.

### Running Benchmarks

```bash
# Quick automated benchmarks
./scripts/run-benchmarks.sh

# Interactive performance test
./scripts/test-performance.sh

# Test with custom dataset
./generate-dataset -count 100000 | ./goose-launcher
```

## Development

```bash
# Build
make build

# Run tests
make test

# Install locally
make install

# Clean
make clean
```

## Contributing

1. Fork the repository
2. Create feature branch: `git checkout -b feature/my-feature`
3. Commit changes: `git commit -m "feat: add my feature"`
4. Push: `git push origin feature/my-feature`
5. Open Pull Request

## License

MIT License - see LICENSE file

## Credits

- Inspired by [fzf](https://github.com/junegunn/fzf)
- Built with [Gio](https://gioui.org)
- Part of the [Goose](https://github.com/sam33r/goosey) launcher ecosystem

