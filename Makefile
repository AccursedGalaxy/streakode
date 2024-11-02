.PHONY: build install clean test release dev-release

# Get the latest git tag
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT_SHA ?= $(shell git rev-parse --short HEAD)
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d %H:%M:%S')

# Build flags - fixed package path and quotes
LDFLAGS := -ldflags="-s -w \
	-X 'main.Version=$(VERSION)' \
	-X 'main.CommitSHA=$(COMMIT_SHA)' \
	-X 'main.BuildTime=$(BUILD_TIME)'"

# Build the binary
build:
	@mkdir -p bin
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/streakode .

# Install globally
install: build
	@echo "Installing streakode..."
	@sudo install -m 755 bin/streakode /usr/local/bin/streakode

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Run tests
test:
	go test ./...

# Development workflow:
dev: clean build install  # Quick rebuild and install for local testing

# Development release (local testing of goreleaser):
dev-release:
	goreleaser release --snapshot --clean

# Create a new release (only run this after git tag):
release-tag:
	@if [ -z "$(TAG)" ]; then echo "Please provide a tag, e.g., make release-tag TAG=v1.0.0"; exit 1; fi
	git tag -a $(TAG) -m "Release $(TAG)"
	git push origin $(TAG)