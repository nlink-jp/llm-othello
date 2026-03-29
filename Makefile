BINARY  := llm-othello
MODULE  := github.com/nlink-jp/$(BINARY)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

GOCACHE     ?= $(HOME)/.cache/go-build
GOMODCACHE  ?= $(HOME)/go/pkg/mod

LDFLAGS := -s -w -X main.version=$(VERSION)

## build: Build binary for the current platform → dist/$(BINARY)
build:
	@mkdir -p dist
	go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY) .

## build-all: Cross-compile for all target platforms → dist/
build-all:
	$(foreach platform,$(PLATFORMS), \
		$(eval OS   := $(word 1,$(subst /, ,$(platform)))) \
		$(eval ARCH := $(word 2,$(subst /, ,$(platform)))) \
		$(eval EXT  := $(if $(filter windows,$(OS)),.exe,)) \
		$(eval OUT  := dist/$(BINARY)_$(OS)_$(ARCH)$(EXT)) \
		GOOS=$(OS) GOARCH=$(ARCH) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) \
		go build -ldflags "$(LDFLAGS)" -o $(OUT) . ;)

## package: Cross-compile and create .zip archives → dist/
package: build-all
	$(foreach platform,$(PLATFORMS), \
		$(eval OS   := $(word 1,$(subst /, ,$(platform)))) \
		$(eval ARCH := $(word 2,$(subst /, ,$(platform)))) \
		$(eval EXT  := $(if $(filter windows,$(OS)),.exe,)) \
		$(eval BIN  := dist/$(BINARY)_$(OS)_$(ARCH)$(EXT)) \
		$(eval ZIP  := dist/$(BINARY)_$(OS)_$(ARCH).zip) \
		zip -j $(ZIP) $(BIN) config.toml ;)

## test: Run tests
test:
	go test ./...

## clean: Remove build artifacts
clean:
	rm -rf dist/

.PHONY: build build-all package test clean
