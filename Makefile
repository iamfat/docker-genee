.PHONY: build install uninstall clean test help build-all build-macos build-linux release prepare-release

# 默认目标
.DEFAULT_GOAL := help

# 变量定义
BINARY_NAME := docker-genee
BUILD_DIR := build

# 从 Go 代码中读取版本号
VERSION := $(shell grep 'Version.*=.*"[^"]*"' cmd/root.go | sed 's/.*Version.*=.*"\([^"]*\)".*/\1/')

# 构建目标平台
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

help: ## 显示帮助信息
	@echo "docker-genee CLI插件构建工具"
	@echo "版本: $(VERSION)"
	@echo ""
	@echo "可用命令:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## 构建当前平台的Go二进制文件
	@echo "构建 $(BINARY_NAME) v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) .

build-all: ## 构建所有平台的二进制文件
	@echo "构建所有平台的 $(BINARY_NAME) v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} go build -ldflags="-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/} .; \
		echo "构建完成: $(BUILD_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}"; \
	done

build-macos: ## 构建macOS版本 (Intel + Apple Silicon)
	@echo "构建macOS版本 v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=amd64 go build -ldflags="-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 go build -ldflags="-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	@echo "macOS版本构建完成: $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64, $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64"

build-linux: ## 构建Linux版本 (AMD64 + ARM64)
	@echo "构建Linux版本 v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -ldflags="-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	@GOOS=linux GOARCH=arm64 go build -ldflags="-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	@echo "Linux版本构建完成: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64, $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64"

install: build ## 安装Docker CLI插件
	@echo "安装Docker CLI插件 v$(VERSION)..."
	@mkdir -p ~/.docker/cli-plugins
	@cp $(BUILD_DIR)/$(BINARY_NAME) ~/.docker/cli-plugins/docker-genee
	@chmod +x ~/.docker/cli-plugins/docker-genee
	@echo "Docker CLI插件安装完成！"
	@echo "现在可以使用 'docker genee' 命令了"
	@echo ""
	@echo "测试命令:"
	@echo "  docker genee --help"
	@echo "  docker genee images"

uninstall: ## 卸载Docker CLI插件
	@echo "卸载Docker CLI插件..."
	@rm -f ~/.docker/cli-plugins/docker-genee
	@echo "Docker CLI插件卸载完成！"

clean: ## 清理构建文件
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR)
	@echo "构建目录已清理: $(BUILD_DIR)"

test: ## 运行测试
	@echo "运行测试..."
	@go test ./...

deps: ## 下载依赖
	@echo "下载Go模块依赖..."
	@go mod download
	@go mod tidy

fmt: ## 格式化代码
	@echo "格式化代码..."
	@go fmt ./...

lint: ## 代码检查
	@echo "代码检查..."
	@go vet ./...

release: build-all ## 构建发布版本
	@echo "构建发布版本 v$(VERSION) 完成！"
	@echo "生成的文件:"
	@ls -la $(BUILD_DIR)/

prepare-release: ## 准备发布版本（使用发布脚本）
	@echo "准备发布版本 v$(VERSION)..."
	@./scripts/release.sh

tag-release: ## 创建并推送版本标签
	@echo "创建版本标签 v$(VERSION)..."
	@git tag v$(VERSION)
	@git push origin v$(VERSION)
	@echo "版本标签 v$(VERSION) 已创建并推送"

show-builds: ## 显示构建结果
	@echo "构建结果 (版本: $(VERSION)):"
	@if [ -d "$(BUILD_DIR)" ]; then \
		ls -la $(BUILD_DIR)/; \
	else \
		echo "构建目录不存在，请先运行构建命令"; \
	fi

version: ## 显示当前版本
	@echo "当前版本: $(VERSION)"
