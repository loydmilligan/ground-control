# Ground Control Makefile

BINARY_NAME=gc
BUILD_DIR=.
INSTALL_DIR=$(HOME)/.local/bin
MAN_DIR=/usr/local/share/man/man1
TLDR_DIR=$(HOME)/.local/share/tldr/pages/common
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

.PHONY: build install install-man install-tldr uninstall clean test help

# Build the binary
build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/gc

# Install to ~/.local/bin (in PATH)
install: build
	@mkdir -p $(INSTALL_DIR)
	cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to $(INSTALL_DIR)/$(BINARY_NAME)"
	@echo "Run 'gc --version' to verify"
	@echo ""
	@echo "Optional: 'make install-man' (requires sudo)"
	@echo "Optional: 'make install-tldr'"

# Install man page (requires sudo)
install-man:
	@sudo mkdir -p $(MAN_DIR)
	sudo cp man/gc.1 $(MAN_DIR)/gc.1
	sudo gzip -f $(MAN_DIR)/gc.1
	@echo "Installed man page. Run 'man gc' to view."

# Install tldr page
install-tldr:
	@mkdir -p $(TLDR_DIR)
	cp man/gc.tldr.md $(TLDR_DIR)/gc.md 2>/dev/null || cp ~/.local/share/tldr/pages/common/gc.md $(TLDR_DIR)/gc.md 2>/dev/null || echo "TLDR page already installed"
	@echo "TLDR page installed. Run 'tldr gc' to view."

# Full install with man and tldr
install-all: install install-tldr
	@echo ""
	@echo "Run 'sudo make install-man' to install man page"

# Remove from ~/.local/bin
uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Removed $(BINARY_NAME) from $(INSTALL_DIR)"

# Clean build artifacts
clean:
	rm -f $(BUILD_DIR)/$(BINARY_NAME)

# Run tests
test:
	go test ./...

# Development: build and run
run: build
	./$(BINARY_NAME)

# Show help
help:
	@echo "Ground Control Build Targets:"
	@echo ""
	@echo "  make build       - Build the gc binary"
	@echo "  make install     - Build and install to ~/.local/bin"
	@echo "  make install-man - Install man page (requires sudo)"
	@echo "  make install-tldr- Install tldr page"
	@echo "  make install-all - Install binary + tldr (man needs sudo)"
	@echo "  make uninstall   - Remove from ~/.local/bin"
	@echo "  make clean       - Remove build artifacts"
	@echo "  make test        - Run tests"
	@echo ""
	@echo "After install:"
	@echo "  gc --version     - Verify installation"
	@echo "  gc --help        - Show commands"
	@echo "  man gc           - View manual (after install-man)"
	@echo "  tldr gc          - View tldr (after install-tldr)"
