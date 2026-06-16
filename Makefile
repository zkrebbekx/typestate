.PHONY: test cover lint fmt tidy verify-guard clean help

test: ## Run unit tests with race detector
	go test -race ./...

cover: ## Run tests and open an HTML coverage report
	go test -coverprofile=coverage.out -coverpkg=./... ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "wrote coverage.html"

lint: ## Run golangci-lint (must be installed)
	golangci-lint run

fmt: ## Format code
	gofmt -s -w .
	go run golang.org/x/tools/cmd/goimports@latest -w .

tidy: ## Tidy modules
	go mod tidy

verify-guard: ## Assert the compile-time guard actually fails to compile
	@if go build -tags typestate_compilefail ./... 2>/dev/null; then \
		echo "FAIL: guard_compilefail.go compiled — typestate guard is broken"; \
		exit 1; \
	else \
		echo "OK: missing-required-field cases are rejected at compile time"; \
	fi

clean: ## Remove build artifacts
	rm -rf bin coverage.out coverage.html

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
