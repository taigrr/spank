SHELL := /bin/bash
REPO_ROOT := $(shell cd .. && pwd)
MACOS_DIR := $(shell pwd)
VERSION   := $(shell git -C $(REPO_ROOT) describe --tags --abbrev=0 2>/dev/null || echo "1.0.0")
BUNDLE    := Spank.app
CONTENTS  := $(BUNDLE)/Contents
GO        := $(shell which go || echo /opt/homebrew/bin/go)

.PHONY: all bundle spankd spankui install clean

all: bundle

# ── Build the spankd daemon (no CGO needed) ────────────────────────────────
spankd:
	@echo "→ Building spankd..."
	cd $(REPO_ROOT) && \
	  CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin \
	  $(GO) build -ldflags="-s -w" -o $(MACOS_DIR)/bin/spankd .

# ── Build the SpankUI menu bar app (CGO required for systray) ──────────────
spankui:
	@echo "→ Building SpankUI..."
	CGO_ENABLED=1 GOARCH=arm64 GOOS=darwin \
	  $(GO) build -ldflags="-s -w" -o $(MACOS_DIR)/bin/SpankUI ./cmd/spankui

# ── Assemble the .app bundle ───────────────────────────────────────────────
bundle: spankd spankui
	@echo "→ Assembling $(BUNDLE)..."
	rm -rf $(BUNDLE)
	mkdir -p $(CONTENTS)/MacOS
	mkdir -p $(CONTENTS)/Resources

	cp bin/SpankUI    $(CONTENTS)/MacOS/SpankUI
	cp bin/spankd     $(CONTENTS)/MacOS/spankd

	cp assets/Info.plist  $(CONTENTS)/Info.plist
	@# Substitute version placeholder if needed
	sed -i '' 's/<string>1.0.0<\/string>/<string>$(VERSION)<\/string>/g' $(CONTENTS)/Info.plist

	@if [ -f assets/Spank.icns ]; then \
	  cp assets/Spank.icns $(CONTENTS)/Resources/Spank.icns; \
	  echo "  ✓ Added Spank.icns"; \
	fi

	@# Ad-hoc code sign (no Developer ID required; users right-click → Open on first launch)
	codesign --force --deep --sign - $(BUNDLE)
	@echo "✓ Built $(BUNDLE)"

# ── Install to /Applications ───────────────────────────────────────────────
install: bundle
	@echo "→ Installing to /Applications..."
	rm -rf /Applications/$(BUNDLE)
	cp -r $(BUNDLE) /Applications/$(BUNDLE)
	@echo "✓ Installed. Run: open /Applications/$(BUNDLE)"

# ── Uninstall ──────────────────────────────────────────────────────────────
uninstall:
	rm -rf /Applications/$(BUNDLE)
	@echo "✓ Uninstalled Spank.app"

# ── Clean build artefacts ──────────────────────────────────────────────────
clean:
	rm -rf bin/ $(BUNDLE)
	@echo "✓ Cleaned"

# ── deps: fetch Go dependencies ────────────────────────────────────────────
deps:
	$(GO) mod tidy
