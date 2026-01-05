# Installation Guide

This guide covers multiple ways to install goose-launcher on your machine.

## Requirements

- **Go 1.21 or later** - [Download](https://go.dev/dl/)
- **macOS** - Currently supports macOS only (Gio UI framework)

## Installation Methods

### Method 1: Using `go install` (Recommended)

This is the easiest method and works on any machine with Go installed:

```bash
go install github.com/sam33r/goose-launcher/cmd/goose-launcher@latest
```

The binary will be installed to `$GOPATH/bin` (usually `~/go/bin`). Make sure this directory is in your `PATH`:

```bash
# Add to ~/.bashrc, ~/.zshrc, or ~/.profile
export PATH="$HOME/go/bin:$PATH"
```

**Verify installation:**
```bash
goose-launcher --help
echo -e "Item 1\nItem 2\nItem 3" | goose-launcher
```

### Method 2: Clone and Build from Source

For development or if you want to modify the code:

```bash
# Clone the repository
git clone https://github.com/sam33r/goose-launcher.git
cd goose-launcher

# Build (creates universal macOS binary)
make build-macos

# Or build for current platform only
make build

# Install to /usr/local/bin
sudo make install
```

### Method 3: Using the install script

The repository includes an install script that builds and installs automatically:

```bash
git clone https://github.com/sam33r/goose-launcher.git
cd goose-launcher
./install.sh
```

This will:
1. Check Go installation
2. Build a universal macOS binary (ARM64 + AMD64)
3. Install to `/usr/local/bin`
4. Verify the installation

### Method 4: Download Pre-built Binary (Future)

*Note: Pre-built binaries will be available in GitHub Releases once we tag a version.*

```bash
# Download latest release
curl -L https://github.com/sam33r/goose-launcher/releases/latest/download/goose-launcher-darwin-universal -o goose-launcher

# Make executable
chmod +x goose-launcher

# Move to PATH
sudo mv goose-launcher /usr/local/bin/
```

## Verify Installation

After installation, verify it works:

```bash
# Check version
goose-launcher --help

# Test basic functionality
echo -e "Test 1\nTest 2\nTest 3" | goose-launcher

# Test with more items
seq 1 100 | goose-launcher

# Test with file listing
find . -type f | goose-launcher
```

## Configuration

### Basic Usage

```bash
# Use as drop-in replacement for fzf
ls | goose-launcher

# With flags
echo -e "Item 1\nItem 2" | goose-launcher --exact --height=80
```

### Available Flags

```
--exact, -e              Enable exact matching (vs fuzzy)
--no-sort                Disable sorting of results
--height=N               Set window height percentage (0-100)
--highlight-matches      Highlight matching text (default: true)
--layout=LAYOUT          Layout mode (default, reverse)
```

### Integration with Goose

Add to `~/.config/goose`:

```bash
LAUNCHER_CMD="goose-launcher -e --no-sort --height=100"
```

Or use environment variable:

```bash
export GOOSE_LAUNCHER="goose-launcher -e --no-sort --height=100"
```

## Uninstallation

### If installed via `go install`:

```bash
rm $(which goose-launcher)
# Or
rm ~/go/bin/goose-launcher
```

### If installed via Makefile or install.sh:

```bash
sudo rm /usr/local/bin/goose-launcher
```

## Troubleshooting

### "command not found: goose-launcher"

The installation directory is not in your PATH. Add it:

```bash
# For go install (add to ~/.zshrc or ~/.bashrc)
export PATH="$HOME/go/bin:$PATH"

# For system install
export PATH="/usr/local/bin:$PATH"
```

### "permission denied"

You may need to use `sudo` for system-wide installation:

```bash
sudo make install
# Or
sudo cp goose-launcher /usr/local/bin/
```

### Build errors

Make sure you have the latest Go version:

```bash
go version  # Should be 1.21 or later
```

If you see Gio-related errors, ensure you're on macOS (Gio UI currently requires macOS).

### Performance issues with large datasets

For datasets > 100k items, disable highlighting:

```bash
find / -type f 2>/dev/null | goose-launcher --highlight-matches=false
```

## Development Setup

If you want to contribute or modify the code:

```bash
# Clone
git clone https://github.com/sam33r/goose-launcher.git
cd goose-launcher

# Install dependencies
go mod download

# Build for testing
make build

# Run tests
make test

# Run benchmarks
./scripts/run-benchmarks.sh

# Test with sample data
./generate-dataset -count 10000 | ./goose-launcher
```

## Platform Support

**Currently Supported:**
- macOS (ARM64 and AMD64)

**Planned:**
- Linux (via Gio's Linux support)
- Windows (via Gio's Windows support)

## Next Steps

- Read [BENCHMARKS.md](BENCHMARKS.md) for performance characteristics
- Check [README.md](README.md) for features and architecture
- See [docs/USAGE.md](docs/USAGE.md) for detailed usage guide

## Getting Help

- **Issues**: [GitHub Issues](https://github.com/sam33r/goose-launcher/issues)
- **Discussions**: [GitHub Discussions](https://github.com/sam33r/goose-launcher/discussions)
- **Documentation**: [README.md](README.md)
