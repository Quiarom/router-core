GO ?= go

.PHONY: fmt vet test build
fmt:
	gofmt -w .
vet:
	$(GO) vet ./...
test:
	$(GO) test ./...
build:
	$(GO) build -o router-core ./cmd/router-core
