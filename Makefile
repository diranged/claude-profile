BINARY_NAME ?= claude-profile
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS = -ldflags "-X github.com/diranged/claude-profile-go/internal/cli.Version=$(VERSION)"
GOBIN=$(shell go env GOPATH)/bin

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run tests.
	go test $$(go list ./... | grep -v /cmd) -coverprofile cover.out -covermode=atomic

.PHONY: cover
cover: ## Display test coverage report.
	go tool cover -func cover.out

.PHONY: coverhtml
coverhtml: ## Generate and open HTML coverage report.
	go tool cover -html cover.out

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter.
	"$(GOLANGCI_LINT)" run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes.
	"$(GOLANGCI_LINT)" run --fix

##@ Build

.PHONY: build
build: fmt vet ## Build binary.
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/main.go

.PHONY: install
install: build ## Install binary to GOPATH/bin.
	cp bin/$(BINARY_NAME) $(GOBIN)/$(BINARY_NAME)

.PHONY: vhs
vhs: ## Install VHS (terminal recorder for demo).
	go install github.com/charmbracelet/vhs@latest

.PHONY: demo
demo: build ## Generate demo GIF using VHS.
	export CLAUDE_PROFILES_DIR=$$(mktemp -d /tmp/claude-demo.XXXXXX) && vhs demo.tape && rm -rf "$$CLAUDE_PROFILES_DIR"

.PHONY: clean
clean: ## Remove build artifacts.
	rm -rf bin/ dist/ cover.out docs/demo.gif

##@ Dependencies

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p "$(LOCALBIN)"

GOLANGCI_LINT_VERSION ?= v2.11.4
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT)
$(GOLANGCI_LINT): $(LOCALBIN)
	@[ -f "$(GOLANGCI_LINT)-$(GOLANGCI_LINT_VERSION)" ] && \
		[ "$$(readlink -- "$(GOLANGCI_LINT)" 2>/dev/null)" = "$(GOLANGCI_LINT)-$(GOLANGCI_LINT_VERSION)" ] || { \
	set -e; \
	echo "Downloading golangci-lint $(GOLANGCI_LINT_VERSION)"; \
	rm -f "$(GOLANGCI_LINT)"; \
	GOBIN="$(LOCALBIN)" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	mv "$(LOCALBIN)/golangci-lint" "$(GOLANGCI_LINT)-$(GOLANGCI_LINT_VERSION)"; \
	}; \
	ln -sf "$$(realpath "$(GOLANGCI_LINT)-$(GOLANGCI_LINT_VERSION)")" "$(GOLANGCI_LINT)"
