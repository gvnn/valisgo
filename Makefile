.PHONY: install setup build run dev test test-cov clean migrate-diff migrate-apply db-up db-down

BINARY_NAME := valisgo
BIN_DIR := bin
COVERAGE_FILE := coverage.out

MIGRATION_NAME ?= auto_migration

PG_URL ?= "postgres://user:pass@localhost:5432/valisgo?search_path=public&sslmode=disable"

install:
	go mod download

setup: install
	go get -tool github.com/air-verse/air@latest

build:
	go build -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/server/main.go

dev:
	go tool air

run:
	go run ./cmd/server

test:
	TEST_DB_DRIVER="postgres" TEST_DB_DSN=$(PG_URL) go test ./...

test-cov:
	TEST_DB_DRIVER="postgres" TEST_DB_DSN=$(PG_URL) go test -coverprofile=$(COVERAGE_FILE) ./...
	go tool cover -func=$(COVERAGE_FILE)

clean:
	rm -rf $(BIN_DIR)/ $(COVERAGE_FILE)

migrate-diff:
	atlas migrate diff $(MIGRATION_NAME) --env postgres

migrate-apply:
	atlas migrate apply --env postgres --url $(PG_URL)
