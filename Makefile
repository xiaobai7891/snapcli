VERSION := 1.0.0
BUILD_DIR := build
BINARY_NAME := snapcli

.PHONY: all build build-windows build-darwin clean deps

all: build

# 安装依赖
deps:
	go mod download
	go mod tidy

# 当前平台构建
build: deps
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/snapcli

# Windows 构建 (64位)
build-windows: deps
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -H windowsgui" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/snapcli

# Windows 构建 (带控制台，用于调试)
build-windows-debug: deps
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-windows-debug.exe ./cmd/snapcli

# macOS 构建 (Intel)
build-darwin-amd64: deps
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/snapcli

# macOS 构建 (Apple Silicon)
build-darwin-arm64: deps
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/snapcli

# 全平台构建
build-all: build-windows build-darwin-amd64 build-darwin-arm64

# 清理
clean:
	rm -rf $(BUILD_DIR)

# 运行
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# 安装到系统
install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
