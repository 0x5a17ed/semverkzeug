BINARY_NAME := semverkzeug

VERSION = $(shell go run ./cmd/semverkzeug describe)

PREFIX ?= $(HOME)/.local
DESTDIR ?=
INSTALL ?= install

CONTAINER_COMPOSE ?= docker compose
CONTAINER_COMPOSE_FILE ?= docker/compose.yaml
CONTAINER_TEST_SERVICE ?= test
GO_TEST_BASE_IMAGE ?= docker.io/library/golang:1.26-trixie
GO_TEST_IMAGE ?= localhost/semverkzeug-test:go1.26-trixie
GO_TEST_FLAGS ?= -v ./...

SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c

GOFLAGS_COMMON := -buildvcs=true -trimpath

GOOS ?= linux
GOARCH ?= amd64

SRC := $(shell find . -type f -name '*.go')

dist/$(BINARY_NAME): GOFLAGS = $(GOFLAGS_COMMON) -ldflags "-w -s -X=main.Version=$(VERSION)"
dist/$(BINARY_NAME): $(SRC) go.mod go.sum
	mkdir -p $(@D)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GOAMD64=v2 go build $(GOFLAGS) -o $@ ./cmd/$(BINARY_NAME)

.PHONY: install
install: dist/$(BINARY_NAME)
	$(INSTALL) -d "$(DESTDIR)$(PREFIX)/bin"
	$(INSTALL) -m 0755 "$<" "$(DESTDIR)$(PREFIX)/bin/$(BINARY_NAME)"

.PHONY: clean
clean:
	rm -f dist/$(BINARY_NAME)

.PHONY: test
test:
	go test $(GO_TEST_FLAGS)

.PHONY: test-containerized
test-containerized:
	env GO_TEST_BASE_IMAGE="$(GO_TEST_BASE_IMAGE)" GO_TEST_IMAGE="$(GO_TEST_IMAGE)" GO_TEST_FLAGS="$(GO_TEST_FLAGS)" \
		$(CONTAINER_COMPOSE) -f $(CONTAINER_COMPOSE_FILE) build $(CONTAINER_TEST_SERVICE)
	env GO_TEST_BASE_IMAGE="$(GO_TEST_BASE_IMAGE)" GO_TEST_IMAGE="$(GO_TEST_IMAGE)" GO_TEST_FLAGS="$(GO_TEST_FLAGS)" \
		$(CONTAINER_COMPOSE) -f $(CONTAINER_COMPOSE_FILE) run --rm $(CONTAINER_TEST_SERVICE)
