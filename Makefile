APP_NAME := moonbase
BUILD_DIR := bin
MAIN := cmd/moonbase/main.go

.PHONY: run build test clean install release

run:
	go run $(MAIN)

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN)

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)

install: build
	cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/$(APP_NAME)

release:
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(MAIN)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(MAIN)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(MAIN)
