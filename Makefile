GO ?= go
GOFMT ?= gofmt

.PHONY: fmt test dev gateway

fmt:
	$(GOFMT) -w $$(find . -name '*.go')

test:
	$(GO) test ./...

dev:
	$(GO) run ./cmd/dev-server

gateway:
	$(GO) run ./cmd/investigation-gateway
