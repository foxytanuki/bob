GO ?= go

.PHONY: build build-binaries fmt

build:
	$(GO) build ./...

build-binaries:
	mkdir -p bin
	$(GO) build -o bin/bob ./cmd/bob
	$(GO) build -o bin/bobd ./cmd/bobd

fmt:
	$(GO) fmt ./...
