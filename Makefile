SHELL := /bin/bash

APP_NAME := avito_test
BIN := ./bin/$(APP_NAME)
PKG := ./...
GO := go

LOAD_BASE ?= http://localhost:18080
LOAD_VUS ?= 10
LOAD_RPS ?= 5
LOAD_DURATION ?= 5m
LOAD_RESULTS ?= loadtest/results

TEST_DB_HOST ?= localhost
TEST_DB_PORT ?= 55432
TEST_DB_USER ?= test_user
TEST_DB_PASSWORD ?= test_pass
TEST_DB_NAME ?= appdb_test

.PHONY: all setup build run test clean \
        up down logs status restart \
        load-up load-down load-test-all \
        test-db-up test-db-down \
        test-unit test-integration test-e2e test-all

all: setup build

setup:
	@if [ ! -f .env ]; then \
		echo "Creating .env from .env.example"; \
		cp .env.example .env; \
	fi

build:
	$(GO) build -o $(BIN) ./cmd/server

run:
	$(GO) run ./cmd/server

test:
	$(GO) test -race -count=1 $(PKG)

clean:
	rm -rf ./bin

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f

status:
	docker compose ps

restart:
	docker compose restart api

load-up:
	docker compose up -d --build db_load migrations_load api_load

load-down:
	docker compose down --remove-orphans --timeout 10

load-test-all: load-up
	@echo "Running load tests against $(LOAD_BASE)"
	@mkdir -p $(LOAD_RESULTS)
	$(GO) run ./cmd/loadtest -base $(LOAD_BASE) -profile baseline -vus $(LOAD_VUS) -rps $(LOAD_RPS) -duration $(LOAD_DURATION) -out $(LOAD_RESULTS)

test-db-up:
	@echo "Starting test database..."
	docker compose -f docker-compose.test.yml up -d db_test
	@echo "Waiting for database to be ready..."
	docker compose -f docker-compose.test.yml up migrations_test
	@echo "Test database is ready"

test-db-down:
	docker compose -f docker-compose.test.yml down

test-integration: test-db-up
	@echo "Running Integration Tests"
	TEST_DB_HOST=$(TEST_DB_HOST) \
	TEST_DB_PORT=$(TEST_DB_PORT) \
	TEST_DB_USER=$(TEST_DB_USER) \
	TEST_DB_PASSWORD=$(TEST_DB_PASSWORD) \
	TEST_DB_NAME=$(TEST_DB_NAME) \
	$(GO) test -v ./internal/integration/... -timeout=5m
	@$(MAKE) test-db-down

test-e2e: test-db-up
	@echo "=== Running E2E Tests ==="
	TEST_DB_HOST=$(TEST_DB_HOST) \
	TEST_DB_PORT=$(TEST_DB_PORT) \
	TEST_DB_USER=$(TEST_DB_USER) \
	TEST_DB_PASSWORD=$(TEST_DB_PASSWORD) \
	TEST_DB_NAME=$(TEST_DB_NAME) \
	$(GO) test -v ./internal/e2e/... -timeout=10m
	@$(MAKE) test-db-down

test-all: test-db-up
	@echo "Running All Tests"
	TEST_DB_HOST=$(TEST_DB_HOST) \
	TEST_DB_PORT=$(TEST_DB_PORT) \
	TEST_DB_USER=$(TEST_DB_USER) \
	TEST_DB_PASSWORD=$(TEST_DB_PASSWORD) \
	TEST_DB_NAME=$(TEST_DB_NAME) \
	$(GO) test -v ./internal/integration/... ./internal/e2e/... -timeout=10m
	@$(MAKE) test-db-down

ci: test-all

dev-all: up load-up test-db-up
	@echo "All environments running:"
	@echo "- App: http://localhost:8080"
	@echo "- Load test: http://localhost:18080"
	@echo "- Test DB: $(TEST_DB_HOST):$(TEST_DB_PORT)"

down-all: down load-down test-db-down
	@echo "All environments stopped"

lint:
	@golangci-lint run --timeout=3m

fmt:
	@gofumpt -w .
	@goimports -w .

tidy:
	$(GO) mod tidy

code-check: tidy fmt lint
	@echo "Code quality check passed"
