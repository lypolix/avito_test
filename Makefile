SHELL := /bin/bash

APP_NAME := avito_test
BIN := ./bin/$(APP_NAME)
PKG := ./...
GO := go

.PHONY: all tidy fmt lint build run test test-integration up down logs setup dev clean migrate-down

all: setup tidy fmt lint build 

setup:
	@if [ ! -f .env ]; then \
		echo "Creating .env file from .env.example..."; \
		cp .env.example .env; \
		echo " .env file created. Please review the configuration if needed."; \
	else \
		echo " .env file already exists"; \
	fi

tidy:
	$(GO) mod tidy

fmt: 
	@gofumpt -w .
	@goimports -w .

lint: 
	@golangci-lint run --timeout=3m

build:
	$(GO) build -o $(BIN) ./cmd/server

run:
	$(GO) run ./cmd/server

test:
	$(GO) test -race -count=1 $(PKG)

test-integration: up
	@echo "Waiting for services to be ready..."
	@sleep 5
	$(GO) test -v ./tests/integration -tags=integration -timeout=60s
	@$(MAKE) down

up:
	docker compose up -d --build	

down:
	docker compose down 

down-clean:
	docker compose down -v

logs:
	docker compose logs -f

migrate-down:
	docker compose run --rm migrations ./migrate -path ./migrations -database "${DATABASE_URL}" down 1

dev: setup up logs

dev-clean: setup down-clean up logs

clean: down-clean
	rm -rf ./bin

status:
	docker compose ps

restart:
	docker compose restart api