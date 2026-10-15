help:  ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

fmt:  ## Format the code
	@goimports -w .
	@gofmt -s -w .


test: ## Run tests and display coverage
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out
	@rm -f coverage.out

# envs/integration.env overrides envs/.env (godotenv: the first listed file wins).
test-integration: ## Run integration tests (needs Docker)
	@go tool godotenv -f envs/integration.env,envs/.env \
		go test -tags integration -count=1 ./test/integration/...

gen: ## Generate code: sqlc
	@sqlc generate
	@mockery

lint: ## Run linters
	@golangci-lint run -v --timeout=5m ./...

infra-up: ## Run infra
	@docker compose up db jaeger -d

rebuild-up: ## Rebuild the image and run the application
	@docker compose up -d --build

up: ## Run the application
	@docker compose up -d

down: ## Stop the application and infra
	@docker compose down