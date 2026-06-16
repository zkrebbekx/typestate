.PHONY: test lint fmt tidy install build clean help

build: ## Build the generator binary into ./bin
	go build -o bin/typestate .

install: ## Install the generator onto $GOBIN / $GOPATH/bin
	go install .

test: ## Run generator tests with race detector
	go test -race ./...

lint: ## Run golangci-lint (must be installed)
	golangci-lint run

fmt: ## Format code
	gofmt -s -w .

tidy: ## Tidy modules
	go mod tidy

clean: ## Remove build artifacts
	rm -rf bin

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'
