.PHONY: all build run test clean demo install

BINARY     := hotreload.exe
BUILD_DIR  := bin
SERVER_BIN := bin\testserver.exe

all: build

## build: Compile the hotreload binary
build:
	@if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)
	go build -o $(BUILD_DIR)\$(BINARY) .
	@echo Built $(BUILD_DIR)\$(BINARY)

## install: Install hotreload to GOPATH/bin
install:
	go install .

## test: Run all unit tests
test:
	go test ./... -v -race -timeout 30s

## test-short: Run tests without race detector
test-short:
	go test ./... -timeout 30s

## demo: Build hotreload and run it against testserver
demo: build
	@echo Starting hot reload demo...
	@echo Edit testserver\main.go and save to trigger a rebuild.
	@echo Visit http://localhost:8080 to see the server.
	$(BUILD_DIR)\$(BINARY) ^
		--root .\testserver ^
		--build "go build -o .\$(SERVER_BIN) .\testserver" ^
		--exec ".\$(SERVER_BIN)" ^
		--log-level debug

## clean: Remove build artifacts
clean:
	if exist $(BUILD_DIR) rmdir /S /Q $(BUILD_DIR)

## help: Show this help
help:
	@echo Available targets: build install test test-short demo clean
