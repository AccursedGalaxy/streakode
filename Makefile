.PHONY: build install clean test release dev-release downgrade

# Get the latest git tag
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
COMMIT_SHA ?= $(shell git rev-parse --short HEAD)
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d %H:%M:%S')

# Build flags - fixed package path and quotes
LDFLAGS := -ldflags="-s -w \
	-X 'main.Version=${VERSION}' \
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
	@if [ ! -e /usr/local/bin/sk ] || [ ! -L /usr/local/bin/sk ] || [ ! "$$(readlink /usr/local/bin/sk)" = "/usr/local/bin/streakode" ]; then \
		echo "Creating 'sk' alias..."; \
		sudo ln -sf /usr/local/bin/streakode /usr/local/bin/sk; \
	else \
		echo "'sk' alias already exists and points to streakode"; \
	fi
	@echo "✨ Installation complete! You can now use 'streakode' or 'sk' to run the CLI."

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

# Create a release for multiple platforms
release:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/streakode-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/streakode-darwin-arm64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/streakode-linux-amd64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/streakode-windows-amd64.exe

# Downgrade to version 1.5.4
downgrade:
	@git checkout tags/v1.5.5
	@make build
	@echo "Downgraded to version 1.5.5. Please run 'make install' to apply the changes."
