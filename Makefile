# Database Stress Test Service Makefile
# PipeOps TCP/UDP Port Testing

.PHONY: help build run test clean docker-build docker-run docker-compose-up docker-compose-down deps fmt lint vet check install dev prod logs

# Default target
.DEFAULT_GOAL := help

# Variables
APP_NAME := stress-test
BINARY_NAME := stress-test
DOCKER_IMAGE := tcp-stress-test
DOCKER_TAG := latest
GO_VERSION := 1.21
PORT := 8080

# Colors for output
RED := \033[31m
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
MAGENTA := \033[35m
CYAN := \033[36m
WHITE := \033[37m
RESET := \033[0m

## help: Show this help message
help:
	@echo "$(CYAN)Database Stress Test Service - Available Commands:$(RESET)"
	@echo ""
	@echo "$(GREEN)Development:$(RESET)"
	@echo "  make build          - Build the application binary"
	@echo "  make run            - Run the application locally"
	@echo "  make dev            - Run in development mode with auto-reload"
	@echo "  make test           - Run all tests"
	@echo "  make deps           - Download and tidy Go modules"
	@echo ""
	@echo "$(GREEN)Code Quality:$(RESET)"
	@echo "  make fmt            - Format Go code"
	@echo "  make lint           - Run linter (golangci-lint)"
	@echo "  make vet            - Run go vet"
	@echo "  make check          - Run fmt, vet, and lint"
	@echo ""
	@echo "$(GREEN)Docker:$(RESET)"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-run     - Run Docker container"
	@echo "  make docker-push    - Push Docker image to registry"
	@echo ""
	@echo "$(GREEN)Docker Compose:$(RESET)"
	@echo "  make compose-up     - Start all services with Docker Compose"
	@echo "  make compose-down   - Stop all services"
	@echo "  make compose-logs   - Show logs from all services"
	@echo "  make compose-dev    - Start services in development mode"
	@echo ""
	@echo "$(GREEN)Database Operations:$(RESET)"
	@echo "  make db-up          - Start only database services"
	@echo "  make db-down        - Stop database services"
	@echo "  make db-clean       - Remove database volumes"
	@echo ""
	@echo "$(GREEN)Utilities:$(RESET)"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make install        - Install development tools"
	@echo "  make logs           - Show application logs"
	@echo "  make health         - Check service health"
	@echo ""

## build: Build the application binary
build:
	@echo "$(BLUE)Building $(APP_NAME)...$(RESET)"
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags='-w -s -extldflags "-static"' \
		-a -installsuffix cgo \
		-o $(BINARY_NAME) ./main.go
	@echo "$(GREEN)Build complete: $(BINARY_NAME)$(RESET)"

## run: Run the application locally
run:
	@echo "$(BLUE)Starting $(APP_NAME)...$(RESET)"
	@echo "$(YELLOW)Dashboard: http://localhost:$(PORT)$(RESET)"
	@echo "$(YELLOW)API Health: http://localhost:$(PORT)/api/health$(RESET)"
	@echo "$(YELLOW)Press Ctrl+C to stop$(RESET)"
	@go run main.go

## dev: Run in development mode with auto-reload
dev:
	@echo "$(BLUE)Starting $(APP_NAME) in development mode...$(RESET)"
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "$(YELLOW)Installing air for auto-reload...$(RESET)"; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

## test: Run all tests
test:
	@echo "$(BLUE)Running tests...$(RESET)"
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Tests complete. Coverage report: coverage.html$(RESET)"

## deps: Download and tidy Go modules
deps:
	@echo "$(BLUE)Downloading dependencies...$(RESET)"
	@go mod download
	@go mod tidy
	@echo "$(GREEN)Dependencies updated$(RESET)"

## fmt: Format Go code
fmt:
	@echo "$(BLUE)Formatting code...$(RESET)"
	@go fmt ./...
	@echo "$(GREEN)Code formatted$(RESET)"

## lint: Run linter
lint:
	@echo "$(BLUE)Running linter...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "$(YELLOW)Installing golangci-lint...$(RESET)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi

## vet: Run go vet
vet:
	@echo "$(BLUE)Running go vet...$(RESET)"
	@go vet ./...
	@echo "$(GREEN)Vet checks passed$(RESET)"

## check: Run fmt, vet, and lint
check: fmt vet lint
	@echo "$(GREEN)All code quality checks passed$(RESET)"

## docker-build: Build Docker image
docker-build:
	@echo "$(BLUE)Building Docker image...$(RESET)"
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "$(GREEN)Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)$(RESET)"

## docker-run: Run Docker container
docker-run: docker-build
	@echo "$(BLUE)Running Docker container...$(RESET)"
	@echo "$(YELLOW)Dashboard: http://localhost:$(PORT)$(RESET)"
	@docker run --rm -p $(PORT):$(PORT) --name $(APP_NAME) $(DOCKER_IMAGE):$(DOCKER_TAG)

## docker-push: Push Docker image to registry
docker-push: docker-build
	@echo "$(BLUE)Pushing Docker image...$(RESET)"
	@docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo "$(GREEN)Docker image pushed$(RESET)"

## compose-up: Start all services with Docker Compose
compose-up:
	@echo "$(BLUE)Starting all services...$(RESET)"
	@docker-compose up -d
	@echo "$(GREEN)Services started$(RESET)"
	@echo "$(YELLOW)Dashboard: http://localhost:$(PORT)$(RESET)"
	@echo "$(YELLOW)Use 'make compose-logs' to view logs$(RESET)"

## compose-down: Stop all services
compose-down:
	@echo "$(BLUE)Stopping all services...$(RESET)"
	@docker-compose down
	@echo "$(GREEN)Services stopped$(RESET)"

## compose-logs: Show logs from all services
compose-logs:
	@docker-compose logs -f

## compose-dev: Start services in development mode
compose-dev:
	@echo "$(BLUE)Starting services in development mode...$(RESET)"
	@docker-compose -f docker-compose.yml up -d --build
	@echo "$(GREEN)Development services started$(RESET)"

## db-up: Start only database services
db-up:
	@echo "$(BLUE)Starting database services...$(RESET)"
	@docker-compose up -d mysql postgres mongo redis mssql
	@echo "$(GREEN)Database services started$(RESET)"
	@echo "$(YELLOW)Waiting for databases to be ready...$(RESET)"
	@sleep 30

## db-down: Stop database services
db-down:
	@echo "$(BLUE)Stopping database services...$(RESET)"
	@docker-compose stop mysql postgres mongo redis mssql
	@echo "$(GREEN)Database services stopped$(RESET)"

## db-clean: Remove database volumes
db-clean: db-down
	@echo "$(RED)Removing database volumes...$(RESET)"
	@docker-compose down -v
	@docker volume prune -f
	@echo "$(GREEN)Database volumes removed$(RESET)"

## clean: Clean build artifacts
clean:
	@echo "$(BLUE)Cleaning build artifacts...$(RESET)"
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@go clean -cache -modcache -testcache
	@echo "$(GREEN)Clean complete$(RESET)"

## install: Install development tools
install:
	@echo "$(BLUE)Installing development tools...$(RESET)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/cosmtrek/air@latest
	@echo "$(GREEN)Development tools installed$(RESET)"

## logs: Show application logs
logs:
	@if docker ps --format "table {{.Names}}" | grep -q "$(APP_NAME)"; then \
		docker logs -f $(APP_NAME); \
	else \
		echo "$(RED)Application container not running$(RESET)"; \
	fi

## health: Check service health
health:
	@echo "$(BLUE)Checking service health...$(RESET)"
	@curl -f http://localhost:$(PORT)/api/health || echo "$(RED)Service not responding$(RESET)"

## monitor: Start monitoring stack (Prometheus + Grafana)
monitor:
	@echo "$(BLUE)Starting monitoring stack...$(RESET)"
	@docker-compose --profile monitoring up -d
	@echo "$(GREEN)Monitoring started$(RESET)"
	@echo "$(YELLOW)Prometheus: http://localhost:9090$(RESET)"
	@echo "$(YELLOW)Grafana: http://localhost:3000 (admin/StrongPassword123!)$(RESET)"

## prod: Start production environment
prod:
	@echo "$(BLUE)Starting production environment...$(RESET)"
	@docker-compose --profile production up -d
	@echo "$(GREEN)Production environment started$(RESET)"

## backup: Backup database volumes
backup:
	@echo "$(BLUE)Creating database backup...$(RESET)"
	@mkdir -p backups/$(shell date +%Y%m%d_%H%M%S)
	@docker-compose exec mysql mysqldump -u root -pStrongPassword123! --all-databases > backups/$(shell date +%Y%m%d_%H%M%S)/mysql_backup.sql
	@docker-compose exec postgres pg_dumpall -U postgres > backups/$(shell date +%Y%m%d_%H%M%S)/postgres_backup.sql
	@echo "$(GREEN)Database backup complete$(RESET)"

## stress: Run a quick stress test
stress:
	@echo "$(BLUE)Running quick stress test...$(RESET)"
	@curl -X POST http://localhost:$(PORT)/api/test \
		-H "Content-Type: application/json" \
		-d '{"duration": 30, "concurrency": 5, "query_type": "simple"}'

## benchmark: Run benchmark test
benchmark:
	@echo "$(BLUE)Running benchmark test...$(RESET)"
	@curl -X POST http://localhost:$(PORT)/api/benchmark \
		-H "Content-Type: application/json" \
		-d '{"database": "MySQL", "query": "SELECT 1", "iterations": 100}'

## status: Check database connection status
status:
	@echo "$(BLUE)Checking database status...$(RESET)"
	@curl -s http://localhost:$(PORT)/api/status | jq '.' || echo "$(RED)Service not responding$(RESET)"

## setup: Complete setup for development
setup: deps install
	@echo "$(GREEN)Setup complete! Ready for development.$(RESET)"
	@echo "$(YELLOW)Next steps:$(RESET)"
	@echo "  1. Copy .env.example to .env and configure databases"
	@echo "  2. Run 'make compose-up' to start all services"
	@echo "  3. Visit http://localhost:$(PORT) for the dashboard"

## ci: Run CI pipeline locally
ci: deps check test build
	@echo "$(GREEN)CI pipeline completed successfully$(RESET)"

# Pattern rule for any undefined target
%:
	@echo "$(RED)Unknown target: $@$(RESET)"
	@echo "$(YELLOW)Run 'make help' to see available commands$(RESET)"
