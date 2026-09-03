GO ?= go
PKG := github.com/Quiarom/router-core/cmd/router-core
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: fmt vet test build release
fmt:
	gofmt -w .
vet:
	$(GO) vet ./...
test:
	$(GO) test ./...
build:
	$(GO) build -ldflags="-X main.version=$(VERSION)" -o router-core ./cmd/router-core
release:
	$(GO) build -ldflags="-X main.version=$(VERSION) -s -w" -trimpath -o dist/router-core ./cmd/router-core
