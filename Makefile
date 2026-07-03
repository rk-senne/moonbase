APP_NAME := moonbase
BUILD_DIR := bin
MAIN := ./cmd/moonbase

VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: run build test clean install setup release lint coverage

run:
	go run $(MAIN)

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN)

test:
	go test ./...

lint:
	go vet ./...
	go run ./cmd/moonbase lint

coverage:
	go test ./... -coverprofile=coverage.out -timeout 60s
	go tool cover -func=coverage.out | tail -1
	go tool cover -html=coverage.out -o coverage.html
	@echo "→ Open coverage.html for details"

clean:
	rm -rf $(BUILD_DIR)

setup: build
	mkdir -p $(HOME)/.local/bin
	ln -sf $(CURDIR)/$(BUILD_DIR)/$(APP_NAME) $(HOME)/.local/bin/$(APP_NAME)
	@echo "✓ moonbase linked to ~/.local/bin/moonbase"
	@echo "  Make sure ~/.local/bin is in your PATH"

install: build
	cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/$(APP_NAME)

release:
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(MAIN)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(MAIN)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(MAIN)
