BINARY_NAME=docktail
VERSION=0.1.0
BUILD_DIR=bin

.PHONY: build clean run test lint

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) .

run: build
	$(BUILD_DIR)/$(BINARY_NAME)

test:
	go test ./... -v

lint:
	golangci-lint run

clean:
	rm -rf $(BUILD_DIR)

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)
