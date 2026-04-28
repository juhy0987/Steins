SHELL := /bin/bash

GO        ?= go
BIN_DIR   := bin
API_BIN   := $(BIN_DIR)/api
SEED_BIN  := $(BIN_DIR)/seed

DATA_DIR  ?= ./data/manga
ADDR      ?= :8080

.PHONY: all build api seed run seed-data test lint vet clean tidy

all: build

build: api seed

api:
	$(GO) build -o $(API_BIN) ./cmd/api

seed:
	$(GO) build -o $(SEED_BIN) ./cmd/seed

run: api
	$(API_BIN) -addr $(ADDR) -data $(DATA_DIR) -pretty

seed-data: seed
	$(SEED_BIN) -data $(DATA_DIR)

test:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed"; exit 1; }
	golangci-lint run

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BIN_DIR)
