BINARY_NAME := semverkzeug

VERSION = $(shell go run ./cmd/semverkzeug describe --no-prefix)

SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c

GOFLAGS_COMMON := -buildvcs=true -trimpath

SRC := $(shell find . -type f -name '*.go')

dist/$(BINARY_NAME): GOFLAGS = $(GOFLAGS_COMMON) -ldflags "-w -s -X=github.com/0x5a17ed/semverkzeug/pkg/version.Version=$(VERSION)"
dist/$(BINARY_NAME): $(SRC) go.mod go.sum
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v2 go build $(GOFLAGS) -o $@ ./cmd/$(BINARY_NAME)

.PHONY: clean
clean:
	rm -f dist/$(BINARY_NAME)

.PHONY: test
test:
	go test -v ./...
