.PHONY: help fmt test test-e2e gen lint infra-up rebuild-up up down

help:  ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

fmt:  ## Format the code
	@goimports -w .
	@gofmt -s -w .


test: ## Run tests and display coverage
	@go tool gotestsum --format pkgname --format-hide-empty-pkg -- -cover ./...

# The env files test-e2e loads; the first listed file wins (godotenv): the tests' own
# settings, then the app config the e2e tests run differently, then the app config itself.
# CI has no envs/app/.env and passes its own: make test-e2e E2E_APP_ENV=...
E2E_APP_ENV ?= envs/app/.env
E2E_ENV_FILES = envs/tests/e2e.env,envs/app/e2e.env,$(E2E_APP_ENV)

test-e2e: ## Run e2e tests (needs Docker)
	@go tool godotenv -f $(E2E_ENV_FILES) \
		go tool gotestsum --format testname -- -tags e2e -count=1 ./test/e2e/...

gen: ## Generate code: sqlc
	@sqlc generate
	@mockery

lint: ## Run linters
	@golangci-lint run -v --timeout=5m --build-tags e2e ./...

infra-up: ## Run infra
	@docker compose up db jaeger -d

rebuild-up: ## Rebuild the image and run the application
	@docker compose up -d --build

up: ## Run the application
	@docker compose up -d

down: ## Stop the application and infra
	@docker compose down