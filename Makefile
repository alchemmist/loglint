.PHONY: build lint vet fmt fmt-check staticcheck plugin golangci-lint-analyzer golangci-lint-rest test test-cover clean

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
GOTESTSUM := $(GOBIN)/gotestsum
GOFUMPT := $(GOBIN)/gofumpt

GOLANGCI_LINT_VERSION ?= latest
STATICCHECK_VERSION ?= latest
GOTESTSUM_VERSION ?= latest
GOFUMPT_VERSION ?= latest

build:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/loglint/

plugin:
	$(GOBUILD) -buildmode=plugin -o $(PLUGIN_NAME) ./plugin/

check: vet fmt-check golangci-lint staticcheck

vet:
	go vet $(GO_PACKAGES)

fmt:
	$(GOFUMPT) version > /dev/null 2>&1 || $(GOCMD) install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)
	@gofmt -w $$(find . -type f -name '*.go' -not -path './.cache/*' -not -path './.git/*' -not -path './vendor/*')
	@$(GOFUMPT) -w -extra $$(find . -type f -name '*.go' -not -path './.cache/*' -not -path './.git/*' -not -path './vendor/*')
	$(GOLANGCI_LINT) run --fix ./...  > /dev/null 2>&1 || true

fmt-check:
	@unformatted="$$(gofmt -l $$(find . -type f -name '*.go' -not -path './.cache/*' -not -path './.git/*' -not -path './vendor/*'))"; \
	test -z "$$unformatted" || (echo "Files not formatted:" && echo "$$unformatted" && exit 1)

$(GOLANGCI_LINT):
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

golangci-lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./...

$(STATICCHECK):
	$(GOCMD) install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)

staticcheck: $(STATICCHECK)
	$(STATICCHECK) $(GO_PACKAGES)

$(GOTESTSUM):
	$(GOCMD) install gotest.tools/gotestsum@$(GOTESTSUM_VERSION)

$(GOFUMPT):
	$(GOCMD) install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)

clean:
	@chmod -R u+w .cache 2>/dev/null || true
	rm -rf $(BINARY_NAME) $(PLUGIN_NAME) coverage.out coverage.html .cache

test: $(GOTESTSUM)
	$(GOTESTSUM) --format short-verbose -- --race -count=1 ./...

test-cover: $(GOTESTSUM)
	$(GOTESTSUM) --format short-verbose -- -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
