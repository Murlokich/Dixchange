.PHONY: help fmt env test test-e2e gen lint infra-up rebuild-up up down

help:  ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

fmt:  ## Format the code
	@goimports -w .
	@gofmt -s -w .

# The app's env file for local runs and docker compose; not committed, it holds the JWT keys.
ENV_FILE ?= envs/app/.env

env: ## Create envs/app/.env from .env.example with new JWT keys (never overwrites)
	@if [ -e $(ENV_FILE) ]; then echo "$(ENV_FILE) already exists, not overwriting it"; exit 1; fi
	@keys=$$(mktemp -d) && trap 'rm -rf "$$keys"' EXIT && \
		openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out "$$keys/private.pem" 2>/dev/null && \
		openssl pkey -in "$$keys/private.pem" -pubout -out "$$keys/public.pem" && \
		sed -e "s|^JWT_PRIVATE_KEY_B64=.*|JWT_PRIVATE_KEY_B64=$$(openssl base64 -A -in "$$keys/private.pem")|" \
			-e "s|^JWT_PUBLIC_KEY_B64=.*|JWT_PUBLIC_KEY_B64=$$(openssl base64 -A -in "$$keys/public.pem")|" \
			envs/app/.env.example > "$$keys/.env" && \
		mv "$$keys/.env" $(ENV_FILE) && \
		echo "created $(ENV_FILE)"


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