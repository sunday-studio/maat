BINARY ?= maat
DIST_DIR ?= dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
VERSION_PKG := github.com/sunday-studio/maat/internal/version
LDFLAGS := -X $(VERSION_PKG).Version=$(VERSION) -X $(VERSION_PKG).Commit=$(COMMIT) -X $(VERSION_PKG).Date=$(DATE)

.PHONY: build check clean release test

test:
	go test ./...

build:
	mkdir -p $(DIST_DIR)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY) ./cmd/maat

release:
	MAAT_BINARY_NAME=$(BINARY) DIST_DIR=$(DIST_DIR) VERSION=$(VERSION) COMMIT=$(COMMIT) DATE=$(DATE) scripts/build-release.sh

check: test build

clean:
	rm -rf $(DIST_DIR)
