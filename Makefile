# Health Hub — root Makefile
# All Go commands run inside server/

SERVER_DIR   := server
GO           := cd $(SERVER_DIR) &&

.PHONY: help test fmt run migrate-up migrate-status smoke lint build clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

test: ## Run all Go tests
	$(GO) go test ./...

fmt: ## Format Go code
	$(GO) gofmt -w .

lint: ## Vet Go code
	$(GO) go vet ./...

build: ## Build the API binary into server/bin/
	$(GO) mkdir -p bin && go build -trimpath -ldflags "-s -w" -o bin/api ./cmd/api

run: ## Run the API server (server/cmd/api)
	$(GO) go run ./cmd/api

migrate-up: ## Run DB migrations up (uses DATABASE_URL_DIRECT)
	$(GO) go run ./cmd/migrate up

migrate-status: ## Show DB migration status
	$(GO) go run ./cmd/migrate status

smoke: ## Run E2E smoke tests (set API_BASE_URL, SMOKE_TOKEN, SMOKE_PROFILE_ID)
	$(GO) go run ./cmd/smoke

clean: ## Remove build artifacts
	rm -rf $(SERVER_DIR)/bin $(SERVER_DIR)/tmp
