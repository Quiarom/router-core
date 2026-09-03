GO ?= go

.PHONY: fmt vet test build frontend-dev frontend-build dev dev-live
fmt:
	gofmt -w .
vet:
	$(GO) vet ./...
test:
	$(GO) test ./...
build:
	$(GO) build -o bin/router-core ./cmd/router-core
	$(GO) build -o bin/router-core-agent ./cmd/router-core-agent
frontend-dev:
	cd frontend && npm run dev
frontend-build:
	cd frontend && npm run build
dev:
	./scripts/dev.sh --mock
dev-live:
	./scripts/dev.sh --live
