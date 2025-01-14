VERSION := $(shell go run ./cmd/semverkzeug describe --no-prefix)

BINARY_NAME := semverkzeug

SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c

GOFLAGS := -buildvcs=true -trimpath -ldflags "-w -s -X=github.com/0x5a17ed/semverkzeug/pkg/version.Version=$(VERSION)"

SRC := $(shell find . -type f \( -name '*.go' \) )

.PHONY:clean
clean:
	rm -f dist/$(BINARY_NAME)

dist/$(BINARY_NAME): $(SRC) go.mod go.sum
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v2 go build $(GOFLAGS) -o $@ ./cmd/$(BINARY_NAME)
