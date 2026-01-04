#!/bin/bash
set -e

echo "=== Goose Native Launcher Installation ==="
echo ""

# Check Go version
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    echo "Install from: https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo "Found Go: $GO_VERSION"

# Build
echo ""
echo "Building goose-launcher..."
make build-macos

# Install
echo ""
echo "Installing to /usr/local/bin..."
sudo cp goose-launcher /usr/local/bin/
sudo chmod +x /usr/local/bin/goose-launcher

# Verify
echo ""
echo "Verifying installation..."
if command -v goose-launcher &> /dev/null; then
    echo "✓ goose-launcher installed successfully"
    echo ""
    echo "Next steps:"
    echo "1. Add to ~/.config/goose:"
    echo "   LAUNCHER_CMD=\"goose-launcher -e --no-sort --height=100\""
    echo ""
    echo "2. Test with: echo -e 'Item 1\\nItem 2' | goose-launcher"
else
    echo "✗ Installation failed"
    exit 1
fi
