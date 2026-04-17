.PHONY: build install test clean

BIN_NAME := goose-launcher
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
INSTALL_PATH := /usr/local/bin

build:
	@echo "Building $(BIN_NAME)..."
	go build $(LDFLAGS) -o $(BIN_NAME) ./cmd/goose-launcher

# Build universal binary for macOS.
# Gio's gl_unix.go requires cgo, so each slice needs a cgo-enabled C compiler
# targeting the right arch. Without -arch, cross-compiling from arm64 → amd64
# (or vice versa) fails with "undefined: Functions" in gioui.org/internal/gl.
build-macos:
	@echo "Building universal macOS binary..."
	CGO_ENABLED=1 CC="clang -arch x86_64" GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_NAME)-amd64 ./cmd/goose-launcher
	CGO_ENABLED=1 CC="clang -arch arm64"  GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_NAME)-arm64 ./cmd/goose-launcher
	lipo -create -output $(BIN_NAME) $(BIN_NAME)-amd64 $(BIN_NAME)-arm64
	@rm -f $(BIN_NAME)-amd64 $(BIN_NAME)-arm64
	@echo "Universal binary created: $(BIN_NAME)"

install: build-macos
	@echo "Installing to $(INSTALL_PATH)..."
	cp $(BIN_NAME) $(INSTALL_PATH)/
	@echo "Installed: $(INSTALL_PATH)/$(BIN_NAME)"

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning..."
	rm -f $(BIN_NAME) $(BIN_NAME)-amd64 $(BIN_NAME)-arm64

help:
	@echo "Goose Native Launcher - Make targets:"
	@echo "  build        Build for current platform"
	@echo "  build-macos  Build universal binary (ARM64 + AMD64)"
	@echo "  install      Build and install to $(INSTALL_PATH)"
	@echo "  test         Run all tests"
	@echo "  clean        Remove built binaries"
