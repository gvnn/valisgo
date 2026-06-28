.PHONY: install setup build run dev test test-cov clean migrate-diff migrate-diff-pg migrate-diff-sqlite migrate-apply-pg migrate-apply-sqlite db-up db-down

BINARY_NAME := valisgo
BIN_DIR := bin
COVERAGE_FILE := coverage.out

MIGRATION_NAME ?= auto_migration

SQLITE_FILE ?= data/valisgo.db

PG_URL ?= "postgres://user:pass@localhost:5432/valisgo?search_path=public&sslmode=disable"
SQLITE_URL ?= "sqlite://$(SQLITE_FILE)"

install:
	go mod download

setup: install
	go get -tool github.com/air-verse/air@latest

build:
	go build -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/server/main.go

dev: $(SQLITE_FILE)
	go tool air

run: $(SQLITE_FILE)
	go run ./cmd/server

test:
	go test ./...

test-cov:
	go test -coverprofile=$(COVERAGE_FILE) ./...
	go tool cover -func=$(COVERAGE_FILE)

clean:
	rm -rf $(BIN_DIR)/ $(COVERAGE_FILE) $(SQLITE_FILE)

$(SQLITE_FILE):
	mkdir -p $(dir $(SQLITE_FILE))
	touch $(SQLITE_FILE)

migrate-diff: migrate-diff-pg migrate-diff-sqlite

migrate-diff-pg:
	atlas migrate diff $(MIGRATION_NAME) --env postgres

migrate-diff-sqlite: $(SQLITE_FILE)
	atlas migrate diff $(MIGRATION_NAME) --env sqlite

migrate-apply-pg:
	atlas migrate apply --env postgres --url $(PG_URL)

migrate-apply-sqlite: $(SQLITE_FILE)
	atlas migrate apply --env sqlite --url $(SQLITE_URL)
