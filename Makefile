GO ?= go
PKG := github.com/Quiarom/router-core/cmd/router-core
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: fmt vet test build release frontend-dev frontend-build dev dev-live
fmt:
	gofmt -w .
vet:
	$(GO) vet ./...
test:
	$(GO) test ./...
build:
	$(GO) build -ldflags="-X main.version=$(VERSION)" -o bin/router-core ./cmd/router-core
	$(GO) build -ldflags="-X main.version=$(VERSION)" -o bin/router-core-agent ./cmd/router-core-agent
release:
	$(GO) build -ldflags="-X main.version=$(VERSION) -s -w" -trimpath -o dist/router-core ./cmd/router-core
frontend-dev:
	cd frontend && npm run dev
frontend-build:
	cd frontend && npm run build
dev:
	./scripts/dev.sh --mock
dev-live:
	./scripts/dev.sh --live
