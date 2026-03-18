.PHONY: build lint vet fmt fmt-check staticcheck plugin golangci-lint-analyzer golangci-lint-rest test test-cover

BINARY_NAME = loglint
PLUGIN_NAME = loglint.so

GOCMD = go
GOBUILD = $(GOCMD) build
GOTEST = $(GOCMD) test
GOMOD = $(GOCMD) mod
TMPDIR ?= /tmp
GOCACHE_DIR := $(TMPDIR)/loglint-cache
export GOCACHE := $(GOCACHE_DIR)/go-build
export GOMODCACHE := $(GOCACHE_DIR)/go-mod
GOBIN := $(shell go env GOPATH)/bin
GO_PACKAGES := ./...

GOLANGCI_LINT := $(GOBIN)/golangci-lint
STATICCHECK := $(GOBIN)/staticcheck

build:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/loglint/

plugin:
	$(GOBUILD) -buildmode=plugin -o $(PLUGIN_NAME) ./plugin/

check: vet fmt-check golangci-lint-analyzer golangci-lint-rest staticcheck

vet:
	go vet $(GO_PACKAGES)

fmt:
	go install mvdan.cc/gofumpt@latest
	gofmt -w .
	gofumpt -w -extra .
	$(GOLANGCI_LINT) run --fix ./...  > /dev/null 2>&1 || true

fmt-check:
	@unformatted="$$(gofmt -l $$(find . -type f -name '*.go' -not -path './.cache/*' -not -path './.git/*' -not -path './vendor/*'))"; \
	test -z "$$unformatted" || (echo "Files not formatted:" && echo "$$unformatted" && exit 1)

$(GOLANGCI_LINT):
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

golangci-lint-analyzer: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./pkg/analyzer/...

golangci-lint-rest: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./cmd/... ./plugin/...

$(STATICCHECK):
	$(GOCMD) install honnef.co/go/tools/cmd/staticcheck@latest

staticcheck: $(STATICCHECK)
	$(STATICCHECK) $(GO_PACKAGES)

test:
	$(GOTEST) --race -count=1 ./...

test-cover:
	$(GOTEST) -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
