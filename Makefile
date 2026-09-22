.DEFAULT_GOAL := help

BIN_DIR = ${PWD}/bin

GOLANGCI_LINT_VERSION ?= v2.13.2
# moq v0.7.1 is built with golang.org/x/tools that can't load packages with Go 1.27+.
# Pinned to the commit from main until the next release.
MOQ_VERSION ?= v0.7.2-0.20260831090446-51bed092a2bd

.PHONY: help tools clean tidy test test-integration lint generate

GREEN=$(shell tput -T xterm setaf 2)
RESET=$(shell tput -T xterm sgr0)

export PATH := ${BIN_DIR}:$(PATH)
export GOBIN := ${BIN_DIR}

tools: ## Install tools
	@echo Installing tools
	CGO_ENABLED=0 go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install github.com/matryer/moq@$(MOQ_VERSION)

clean: ## Run all cleanup tasks
	go clean ./...
	rm -rf $(BIN_DIR)
	rm -rf ./builds

tidy: ## Run go mod tidy for the library and the integration tests modules
	go mod tidy
	cd tests && go mod tidy

test: ## Run unit tests
	go test -count=1 -race -v ./...
	@echo ""
	@echo "${GREEN} All tests passed ✅"
	@echo "${RESET}"

test-integration: ## Run integration tests (requires Docker). Consul image can be set with CONSUL_IMAGE
	cd tests && go test -count=1 -timeout 300s -tags integration -v ./...
	@echo ""
	@echo "${GREEN} All tests passed ✅"
	@echo "${RESET}"

lint: tools ## Run linter
	${BIN_DIR}/golangci-lint --color=always run ./... -v --timeout 15m
	cd tests && ${BIN_DIR}/golangci-lint --color=always run ./... -v --timeout 15m

generate: tools ## Generate mocks
# Because go generate doesn't support subshell we need to call it directly
	${BIN_DIR}/moq -pkg consul -out ./mocks_grpc_test.go $$(go list -f '{{.Dir}}' google.golang.org/grpc/resolver) ClientConn
	go generate -x ./...

help: ## Display help screen
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / \
	{printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
