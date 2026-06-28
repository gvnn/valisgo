.PHONY: install setup build run dev test test-cov clean

BINARY_NAME := valisgo
BIN_DIR := bin
COVERAGE_FILE := coverage.out

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
	go test ./...

test-cov:
	go test -coverprofile=$(COVERAGE_FILE) ./...
	go tool cover -func=$(COVERAGE_FILE)

clean:
	rm -rf $(BIN_DIR)/ $(COVERAGE_FILE) data/valisgo.db
