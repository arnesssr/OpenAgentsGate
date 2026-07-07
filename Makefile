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
	tmp="$$(mktemp -d)"; \
	trap 'rm -rf "$$tmp"' EXIT; \
	XDG_STATE_HOME="$$tmp/state" go run $(PKG) init -config "$$tmp/config.json" >/dev/null; \
	go run $(PKG) check -config "$$tmp/config.json" -action github.create_pr -agent codex -resource repo >/dev/null; \
	go run $(PKG) audit verify -config "$$tmp/config.json"
	install_tmp="$$(mktemp -d)"; \
	trap 'rm -rf "$$install_tmp"' EXIT; \
	mkdir -p "$$install_tmp/bin" "$$install_tmp/work"; \
	GOBIN="$$install_tmp/bin" go install $(PKG); \
	cd "$$install_tmp/work"; \
	XDG_CONFIG_HOME="$$install_tmp/config" XDG_STATE_HOME="$$install_tmp/state" "$$install_tmp/bin/openagentsgate" init >/dev/null; \
	XDG_CONFIG_HOME="$$install_tmp/config" XDG_STATE_HOME="$$install_tmp/state" "$$install_tmp/bin/openagentsgate" config doctor >/dev/null; \
	XDG_CONFIG_HOME="$$install_tmp/config" XDG_STATE_HOME="$$install_tmp/state" "$$install_tmp/bin/openagentsgate" check -action github.create_pr -agent codex -resource repo >/dev/null; \
	XDG_CONFIG_HOME="$$install_tmp/config" XDG_STATE_HOME="$$install_tmp/state" "$$install_tmp/bin/openagentsgate" audit verify >/dev/null

clean:
	rm -rf $(BIN_DIR) dist
