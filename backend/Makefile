# FireMail Makefile

# 变量定义
APP_NAME = firemail
VERSION = 1.0.0
BUILD_DIR = build
BINARY = $(BUILD_DIR)/$(APP_NAME)
MAIN_FILE = cmd/firemail/main.go

# Go相关变量
GO = go
GOCMD = $(GO)
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod

# 构建标志
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"
BUILD_FLAGS = -v $(LDFLAGS)

# 平台相关
GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)

# 默认目标
.PHONY: all
all: clean deps build

# 安装依赖
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# 构建应用
.PHONY: build
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(BUILD_FLAGS) -o $(BINARY) $(MAIN_FILE)

# 构建所有平台
.PHONY: build-all
build-all: clean
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(MAIN_FILE)
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 $(MAIN_FILE)
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(MAIN_FILE)
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(MAIN_FILE)
	
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(MAIN_FILE)

# 运行应用
.PHONY: run
run: build
	@echo "Running $(APP_NAME)..."
	./$(BINARY)

# 开发模式运行
.PHONY: dev
dev:
	@echo "Running in development mode..."
	$(GOCMD) run $(MAIN_FILE)

# 运行测试
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# 运行测试并生成覆盖率报告
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 运行基准测试
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# 代码格式化
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# 代码检查
.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

# 代码静态分析
.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# 安全检查
.PHONY: security
security:
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# 清理构建文件
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# 安装工具
.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOGET) github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Docker相关
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) .
	docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest

.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run -d \
		--name $(APP_NAME) \
		-p 8080:8080 \
		-v $(PWD)/data:/root/data \
		-e DB_PATH=/root/data/firemail.db \
		$(APP_NAME):latest

.PHONY: docker-stop
docker-stop:
	@echo "Stopping Docker container..."
	docker stop $(APP_NAME) || true
	docker rm $(APP_NAME) || true

# 数据库相关
.PHONY: db-migrate
db-migrate:
	@echo "Running database migrations..."
	$(GOCMD) run $(MAIN_FILE) -migrate

.PHONY: db-reset
db-reset:
	@echo "Resetting database..."
	rm -f firemail.db
	$(MAKE) db-migrate

# 生成文档
.PHONY: docs
docs:
	@echo "Generating documentation..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Starting godoc server at http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo "godoc not installed. Install with: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# 代码质量检查
.PHONY: quality
quality: fmt vet lint test

# 完整的CI检查
.PHONY: ci
ci: deps quality test-coverage security

# 发布准备
.PHONY: release
release: clean ci build-all
	@echo "Release $(VERSION) ready in $(BUILD_DIR)/"

# 帮助信息
.PHONY: help
help:
	@echo "FireMail Makefile Commands:"
	@echo ""
	@echo "Build Commands:"
	@echo "  build        - Build the application"
	@echo "  build-all    - Build for all platforms"
	@echo "  clean        - Clean build files"
	@echo ""
	@echo "Development Commands:"
	@echo "  run          - Build and run the application"
	@echo "  dev          - Run in development mode"
	@echo "  deps         - Install dependencies"
	@echo ""
	@echo "Testing Commands:"
	@echo "  test         - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  bench        - Run benchmarks"
	@echo ""
	@echo "Code Quality Commands:"
	@echo "  fmt          - Format code"
	@echo "  vet          - Run go vet"
	@echo "  lint         - Run linter"
	@echo "  security     - Run security checks"
	@echo "  quality      - Run all quality checks"
	@echo ""
	@echo "Docker Commands:"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  docker-stop  - Stop Docker container"
	@echo ""
	@echo "Database Commands:"
	@echo "  db-migrate   - Run database migrations"
	@echo "  db-reset     - Reset database"
	@echo ""
	@echo "Other Commands:"
	@echo "  install-tools - Install development tools"
	@echo "  docs         - Generate documentation"
	@echo "  ci           - Run CI checks"
	@echo "  release      - Prepare release"
	@echo "  help         - Show this help"

# 默认显示帮助
.DEFAULT_GOAL := help
