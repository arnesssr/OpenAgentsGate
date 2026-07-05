APP := openagentsgate
PKG := ./cmd/openagentsgate
BIN_DIR ?= bin
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X github.com/arnesssr/OpenAgentsGate/internal/buildinfo.Version=$(VERSION) -X github.com/arnesssr/OpenAgentsGate/internal/buildinfo.Commit=$(COMMIT) -X github.com/arnesssr/OpenAgentsGate/internal/buildinfo.Date=$(DATE)

.PHONY: all build install test vet verify clean

all: build

build:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP) $(PKG)

install:
	CGO_ENABLED=0 go install -trimpath -ldflags "$(LDFLAGS)" $(PKG)

test:
	go test ./...

vet:
	go vet ./...

verify: test vet
	go run $(PKG) check -config examples/openagentsgate.json -action github.create_pr -agent codex -resource repo
	go run $(PKG) tool git -config examples/openagentsgate.json -- status --short

clean:
	rm -rf $(BIN_DIR) dist
