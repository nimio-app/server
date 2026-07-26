.PHONY: help build run test clean docker-up docker-down migrate-up migrate-down migrate-create

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building..."
	@go build -o bin/api cmd/api/main.go

run: ## Run the application
	@echo "Running..."
	@export PATH="/opt/homebrew/bin:$$PATH" && CGO_ENABLED=0 go run cmd/api/main.go

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

docker-up: ## Start Docker containers
	@echo "Starting Docker containers..."
	@docker-compose up -d

docker-down: ## Stop Docker containers
	@echo "Stopping Docker containers..."
	@docker-compose down

docker-logs: ## View Docker logs
	@docker-compose logs -f

migrate-up: ## Run database migrations up
	@echo "Running migrations up..."
	@migrate -path migrations -database "postgres://nimio:nimio_dev_password@localhost:5432/nimio_db?sslmode=disable" up

migrate-down: ## Rollback last migration
	@echo "Rolling back last migration..."
	@migrate -path migrations -database "postgres://nimio:nimio_dev_password@localhost:5432/nimio_db?sslmode=disable" down 1

migrate-create: ## Create a new migration file (usage: make migrate-create name=create_users_table)
	@if [ -z "$(name)" ]; then echo "Error: name is required. Usage: make migrate-create name=your_migration_name"; exit 1; fi
	@echo "Creating migration: $(name)"
	@migrate create -ext sql -dir migrations -seq $(name)

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

dev: docker-up ## Start development environment
	@echo "Waiting for database to be ready..."
	@sleep 3
	@$(MAKE) run
