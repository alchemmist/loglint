.PHONY: build lint vet fmt fmt-check staticcheck plugin golangci-lint-analyzer golangci-lint-rest

BINARY_NAME = loglint
PLUGIN_NAME = loglint.so

GOCMD = go
GOBUILD = $(GOCMD) build
GOTEST = $(GOCMD) test
GOMOD = $(GOCMD) mod
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
	@test -z "$$(gofmt -l .)" || (echo "Files not formatted:" && gofmt -l . && exit 1)

golangci-lint-analyzer:
	$(GOLANGCI_LINT) run ./pkg/analyzer/...

golangci-lint-rest:
	$(GOLANGCI_LINT) run ./cmd/... ./plugin/...

staticcheck:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	$(STATICCHECK) $(GO_PACKAGES)
