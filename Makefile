.PHONY: build

BINARY_NAME = loglint
PLUGIN_NAME = loglint.so

GOCMD = go
GOBUILD = $(GOCMD) build
GOTEST = $(GOCMD) test
GOVET = $(GOCMD) vet
GOFMT = gofmt
GOMOD = $(GOCMD) mod
GOBIN := $(shell go env GOPATH)/bin
GO_PACKAGES := ./...

GOLANGCI_LINT := $(GOBIN)/golangci-lint
STATICCHECK := $(GOBIN)/staticcheck

build:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/loglint/

lint: vet fmt-check golangci-lint

vet:
	$(GOVET) $(GO_PACKAGES)

fmt:
	$(GOFMT) -w .

fmt-check:
	@test -z "$$($(GOFMT) -l .)" || (echo "Files not formatted:" && $(GOFMT) -l . && exit 1)

golangci-lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOLANGCI_LINT) run $(GO_PACKAGES) 

staticcheck:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	$(STATICCHECK) $(GO_PACKAGES)
