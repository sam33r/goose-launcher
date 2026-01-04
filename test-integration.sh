#!/bin/bash
set -e

echo "=== Goose Launcher Integration Tests ==="

# Build
echo "Building..."
go build -o goose-launcher ./cmd/goose-launcher

# Test 1: Empty input
echo ""
echo "Test 1: Empty input"
echo "" | ./goose-launcher || echo "✓ Correctly exits with empty input"

# Test 2: Plugin format parsing
echo ""
echo "Test 2: Plugin separator parsing (manual test)"
echo "Type 'file' and press Enter to select first match"
echo -e "files   . /home/user/file.txt\nchrome   . https://example.com" | ./goose-launcher || true

# Test 3: Fuzzy search
echo ""
echo "Test 3: Fuzzy search filtering (manual test)"
echo "Type 'doc' to filter, then press Enter"
echo -e "Downloads/file1.txt\nDocuments/notes.md\nDesktop/image.png" | ./goose-launcher || true

echo ""
echo "=== Basic tests complete ==="
echo "Note: Manual interaction tests require user input"
