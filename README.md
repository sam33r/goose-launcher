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

## Quick Start

```bash
# Install
./install.sh

# Test
echo -e "Item 1\nItem 2\nItem 3" | goose-launcher

# Configure Goose
echo 'LAUNCHER_CMD="goose-launcher -e --no-sort --height=100"' >> ~/.config/goose
```

## Documentation

- [Usage Guide](docs/USAGE.md)
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

Current benchmarks (MVP):
- Launch time: ~50ms ✓
- Filter latency: <50ms ✓
- Memory: <30MB (10k items) ✓
- Binary size: 8.3MB (universal) ✓

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

