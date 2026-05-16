.PHONY: build test smoke install lint cross generate proto

GO        ?= go
PKG        := ./...
BIN        := bin/fsdtrace
LDFLAGS    := -s -w
BUILDFLAGS := -trimpath -ldflags="$(LDFLAGS)"

build:
	CGO_ENABLED=1 $(GO) build $(BUILDFLAGS) -o $(BIN) ./cmd/fsdtrace

test:
	$(GO) test -race $(PKG)

smoke:
	./scripts/smoke.sh

install:
	CGO_ENABLED=1 $(GO) install $(BUILDFLAGS) ./cmd/fsdtrace

lint:
	golangci-lint run $(PKG)

# Cross-compiling this project requires target C toolchains because the
# Spring indexer uses tree-sitter via CGO. See scripts/cross-compile.sh.
cross:
	./scripts/cross-compile.sh

generate:
	$(GO) generate $(PKG)
