.PHONY: build run test clean fmt lint

VERSION := 2.0.0
BINARY := lionclaw
GOFLAGS := -tags "fts5" -ldflags="-s -w -X main.version=$(VERSION)"

# 构建
build:
	@echo "🦁 构建 LionClaw $(VERSION)..."
	CGO_ENABLED=1 go build $(GOFLAGS) -o bin/$(BINARY) ./cmd/lionclaw
	@echo "✅ 构建完成: bin/$(BINARY)"
	@ls -lh bin/$(BINARY)

# 运行
run: build
	./bin/$(BINARY) start

# 测试
test:
	CGO_ENABLED=1 go test -v -race -tags "fts5" ./...

# 测试覆盖率
cover:
	CGO_ENABLED=1 go test -coverprofile=coverage.out -tags "fts5" ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ 覆盖率报告: coverage.html"

# 格式化
fmt:
	go fmt ./...
	@echo "✅ 代码已格式化"

# 清理
clean:
	rm -rf bin/ coverage.out coverage.html
	@echo "✅ 已清理"

# 安装依赖
deps:
	go mod tidy
	@echo "✅ 依赖已更新"

# 查看二进制大小
size: build
	@echo "📦 二进制信息:"
	@file bin/$(BINARY)
	@du -h bin/$(BINARY)
