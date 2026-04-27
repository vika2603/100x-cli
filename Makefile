BIN       := bin/100x
PKG       := ./...
# E100X_ENDPOINT_DEFAULT controls the build-time default API endpoint. Set it
# in your shell or CI to bake a value in; leave unset to require runtime
# $E100X_ENDPOINT.
LDFLAGS   := -X github.com/vika2603/100x-cli/internal/version.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev) \
             -X github.com/vika2603/100x-cli/internal/version.Commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo none) \
             -X github.com/vika2603/100x-cli/internal/version.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ) \
             -X github.com/vika2603/100x-cli/internal/config.DefaultEndpoint=$(E100X_ENDPOINT_DEFAULT)

.PHONY: build install test fmt vet lint tidy clean run snapshot release-check

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/100x

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/100x

test:
	go test -race -count=1 $(PKG)

fmt:
	gofmt -s -w .
	goimports -w . 2>/dev/null || true

vet:
	go vet $(PKG)

# Requires `golangci-lint` on PATH (https://golangci-lint.run/usage/install/).
lint:
	golangci-lint run

tidy:
	go mod tidy

clean:
	rm -rf bin

run: build
	./$(BIN) $(ARGS)

# Build a local snapshot via goreleaser (requires goreleaser on PATH).
# Output goes to ./dist/.
snapshot:
	goreleaser release --snapshot --clean

# Validate the .goreleaser.yaml without producing artifacts.
release-check:
	goreleaser check
