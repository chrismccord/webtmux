# WebTmux Makefile
# Builds portable binaries for all standard platforms

VERSION ?= $(shell git describe --tags 2>/dev/null || echo "dev")
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_OPTIONS = -ldflags "-s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT)"

OUTPUT_DIR = ./builds
BINARY_NAME = webtmux

# Platforms to build for (PTY not supported on Windows)
PLATFORMS = \
	linux/amd64 \
	linux/arm64 \
	linux/arm \
	darwin/amd64 \
	darwin/arm64 \
	freebsd/amd64

export CGO_ENABLED=0

.PHONY: all build clean test install cross-compile release help

# Default target
all: build

# Sync resources to bindata (for embedding)
sync-assets:
	@cp -r resources/js/* bindata/static/js/

# Build for current platform
build: sync-assets
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	go build $(BUILD_OPTIONS) -o $(BINARY_NAME) .
	@echo "Done: ./$(BINARY_NAME)"

# Install to GOPATH/bin
install:
	go install $(BUILD_OPTIONS) .

# Run tests
test:
	go test ./...
	go vet ./...

# Clean build artifacts
clean:
	rm -rf $(BINARY_NAME) $(OUTPUT_DIR)

# Cross-compile for all platforms
cross-compile: clean
	@echo "Cross-compiling $(BINARY_NAME) $(VERSION) for all platforms..."
	@mkdir -p $(OUTPUT_DIR)
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d/ -f1); \
		arch=$$(echo $$platform | cut -d/ -f2); \
		output=$(OUTPUT_DIR)/$(BINARY_NAME)-$$os-$$arch; \
		if [ "$$os" = "windows" ]; then output=$$output.exe; fi; \
		echo "  Building $$os/$$arch..."; \
		GOOS=$$os GOARCH=$$arch go build $(BUILD_OPTIONS) -o $$output . || exit 1; \
	done
	@echo "Done! Binaries in $(OUTPUT_DIR)/"
	@ls -lh $(OUTPUT_DIR)/

# Create release archives
release: cross-compile
	@echo "Creating release archives..."
	@mkdir -p $(OUTPUT_DIR)/dist
	@cd $(OUTPUT_DIR) && for f in $(BINARY_NAME)-*; do \
		if [ -f "$$f" ]; then \
			tar -czf dist/$$f.tar.gz $$f; \
		fi; \
	done
	@cd $(OUTPUT_DIR)/dist && sha256sum * > SHA256SUMS
	@echo "Release archives in $(OUTPUT_DIR)/dist/"
	@ls -lh $(OUTPUT_DIR)/dist/

# Copy JS assets to bindata (for development)
assets:
	cp resources/js/webtmux.js bindata/static/js/
	cp resources/js/components/*.js bindata/static/js/components/

# Development build with assets
dev: assets build

help:
	@echo "WebTmux Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make              Build for current platform"
	@echo "  make build        Build for current platform"
	@echo "  make install      Install to GOPATH/bin"
	@echo "  make test         Run tests"
	@echo "  make clean        Remove build artifacts"
	@echo "  make cross-compile Build for all platforms"
	@echo "  make release      Create release archives"
	@echo "  make assets       Copy JS assets to bindata"
	@echo "  make dev          Build with fresh assets"
	@echo "  make help         Show this help"
