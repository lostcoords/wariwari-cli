.PHONY: build test-unit test check fmt fmt-check vet tidy clean sync-spec

BINARY := $(CURDIR)/bin/wariwari
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.DEFAULT_GOAL := check

build:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/wariwari

test-unit:
	go test ./internal/...

test: test-unit

fmt:
	gofmt -s -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Files not formatted:"; gofmt -l .; exit 1)

vet:
	go vet ./...

tidy:
	go mod tidy

check: fmt-check vet test-unit

# Refresh the vendored API contract the tests check commands against.
sync-spec:
	cp ../wariwari/openapi/dist/wariwari.bundled.yaml internal/commands/testdata/wariwari.openapi.yaml

clean:
	rm -rf bin dist
	go clean

