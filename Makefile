GO ?= go
PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin

.PHONY: build build-binaries install fmt

build:
	$(GO) build ./...

build-binaries:
	mkdir -p bin
	$(GO) build -o bin/bob ./cmd/bob
	$(GO) build -o bin/bobd ./cmd/bobd

install: build-binaries
	mkdir -p $(BINDIR)
	install -m 755 bin/bob $(BINDIR)/bob
	install -m 755 bin/bobd $(BINDIR)/bobd

fmt:
	$(GO) fmt ./...
