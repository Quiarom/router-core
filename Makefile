# Gavetero / router-core Makefile.
#
# The user-facing workflow targets `gavetero` (alias `gvt`) as the
# only binary the end user types. The transitional binaries
# (router-core, router-core-agent, router-core-learn) stay
# available for the engineering harness; the desktop build
# continues to use them via Tauri's sidecar config.
#
# Conventions:
#   `make build`        compile the in-tree bin/ output
#   `make install-user` install gvt + gavetero to ~/.local/bin
#   `make test`         go test ./...
#   `make check`        full deterministic check (gofmt + vet + test -race + frontend)
#   `make demo`         scripts/dev.sh --mock (transitional, removed post-rename)
GO ?= go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
DIST ?= bin
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin

# Transitional binaries still owned by router-core.
ROUTER_CORE_CMDS = router-core router-core-agent router-core-learn

.PHONY: build install-user uninstall-user fmt vet test check \
        frontend-install frontend-test frontend-build \
        dev dev-live clean

build:
	@mkdir -p $(DIST)
	$(GO) build -ldflags="-X main.version=$(VERSION) -X github.com/Quiarom/router-core/cmd/gavetero/cmd.version=$(VERSION)" -o $(DIST)/gavetero ./cmd/gavetero
	@ln -sf gavetero $(DIST)/gvt
	@# Transitional binaries, still needed by the engineering harness
	@# and by the desktop sidecar. Not user-facing.
	$(GO) build -ldflags="-X main.version=$(VERSION)" -o $(DIST)/router-core ./cmd/router-core
	$(GO) build -ldflags="-X main.version=$(VERSION)" -o $(DIST)/router-core-agent ./cmd/router-core-agent
	$(GO) build -ldflags="-X main.version=$(VERSION)" -o $(DIST)/router-core-learn ./cmd/router-core-learn
	@echo "Built: $(DIST)/{gavetero,gvt -> gavetero,router-core,router-core-agent,router-core-learn}"

# Install the user-facing binary into ~/.local/bin.
# Does not require sudo. Idempotent: re-running is a no-op.
install-user: build
	@mkdir -p $(BINDIR)
	@install -m 0755 $(DIST)/gavetero $(BINDIR)/gavetero
	@ln -sf gavetero $(BINDIR)/gvt
	@echo "Installed:"
	@echo "  $(BINDIR)/gavetero"
	@echo "  $(BINDIR)/gvt -> gavetero"
	@if echo "$$PATH" | tr ':' '\n' | grep -qx "$(BINDIR)"; then \
		echo ""; \
		echo "Try in a new shell:"; \
		echo "  gvt version"; \
	else \
		echo ""; \
		echo "$(BINDIR) is not on PATH. Add this line to your shell rc:"; \
		echo "  export PATH=\"$(BINDIR):\$$PATH\""; \
		echo "Then open a new shell."; \
	fi

uninstall-user:
	@rm -f $(BINDIR)/gavetero $(BINDIR)/gvt
	@echo "Removed $(BINDIR)/gavetero and $(BINDIR)/gvt"

fmt:
	gofmt -w .

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

# Full deterministic check. No router. No GMI key. No network.
check: fmt vet test frontend-test
	@echo "check OK"

frontend-install:
	cd frontend && npm ci

frontend-test:
	cd frontend && npm test

frontend-build:
	cd frontend && npm run build

# Engineering harness: the transitional binaries, used by the
# Tauri desktop and by the CI golden trace. Not user-facing.
dev:
	./scripts/dev.sh --mock

dev-live:
	./scripts/dev.sh --live

clean:
	rm -rf $(DIST) dist
