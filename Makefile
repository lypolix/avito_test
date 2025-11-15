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

.PHONY: all setup tidy fmt lint build run test \
        up down down-clean logs migrate-down status restart dev dev-clean clean \
        build-loadtest load-test load-test-smoke load-test-baseline load-test-mass load-test-all \
        load-up load-down load-down-clean load-logs load-migrate-down load-restart \
        docker-up load-run-all

all: setup tidy fmt lint build

setup:
	@if [ ! -f .env ]; then \
		echo "Creating .env from .env.example"; \
		cp .env.example .env; \
	else \
		echo ".env already exists"; \
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

clean:
	rm -rf ./bin

up:
	docker compose up -d --build

down:
	docker compose down

down-clean:
	docker compose down -v

logs:
	docker compose logs -f

migrate-down:
	docker compose run --rm migrations ./migrate -path ./migrations -database "$${DATABASE_URL}" down 1

status:
	docker compose ps

restart:
	docker compose restart api

dev: setup up logs
dev-clean: setup down-clean up logs

load-up:
	docker compose up -d --build db_load migrations_load api_load

load-down:
	docker compose down --remove-orphans --timeout 10

load-down-clean:
	docker compose down -v --remove-orphans --timeout 10

load-logs:
	docker compose logs -f api_load db_load migrations_load

load-migrate-down:
	docker compose run --rm migrations_load ./migrate -path ./migrations -database "$${LOAD_DATABASE_URL}" down 1

load-restart:
	docker compose restart api_load

docker-up: up load-up

build-loadtest:
	$(GO) build -o bin/loadtest ./cmd/loadtest

load-test: build-loadtest
	@mkdir -p $(LOAD_RESULTS)
	./bin/loadtest -base $(LOAD_BASE) -profile $(LOAD_PROFILE) -vus $(LOAD_VUS) -rps $(LOAD_RPS) -duration $(LOAD_DURATION) -out $(LOAD_RESULTS)

load-test-smoke: build-loadtest
	@mkdir -p $(LOAD_RESULTS)
	./bin/loadtest -base $(LOAD_BASE) -profile smoke -vus $(LOAD_VUS) -rps $(LOAD_RPS) -duration 1m -out $(LOAD_RESULTS)

load-test-baseline: build-loadtest
	@mkdir -p $(LOAD_RESULTS)
	./bin/loadtest -base $(LOAD_BASE) -profile baseline -vus $(LOAD_VUS) -rps $(LOAD_RPS) -duration $(LOAD_DURATION) -out $(LOAD_RESULTS)

load-test-mass: build-loadtest
	@mkdir -p $(LOAD_RESULTS)
	./bin/loadtest -base $(LOAD_BASE) -profile mass -vus $(LOAD_VUS) -rps $(LOAD_RPS) -duration $(LOAD_DURATION) -out $(LOAD_RESULTS)

load-test-all: build-loadtest
	@echo "==> SMOKE"
	$(MAKE) load-test-smoke
	@echo "==> BASELINE"
	$(MAKE) load-test-baseline
	@echo "==> MASS"
	$(MAKE) load-test-mass
	@echo "==> All load tests finished. Results in $(LOAD_RESULTS)"

load-run-all: load-up load-test-all
	@echo "==> Load tests finished against $(LOAD_BASE)"
