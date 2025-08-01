.PHONY: build run test clean docker-build docker-run docker-stop init-db benchmark help

# 默认目标
all: build

# 构建应用
build:
	@echo "构建应用..."
	go build -o bin/user_system main.go

# 运行应用
run: build
	@echo "启动应用..."
	./bin/user_system

# 开发模式运行
dev:
	@echo "开发模式启动..."
	go run main.go

# 运行测试
test:
	@echo "运行测试..."
	go test ./...

# 清理构建文件
clean:
	@echo "清理构建文件..."
	rm -rf bin/
	go clean

# Docker构建
docker-build:
	@echo "构建Docker镜像..."
	docker build -t user_system .

# Docker运行
docker-run:
	@echo "启动Docker服务..."
	docker-compose up -d

# Docker停止
docker-stop:
	@echo "停止Docker服务..."
	docker-compose down

# 初始化数据库
init-db:
	@echo "初始化数据库..."
	go run scripts/init_db.go

# 性能测试
benchmark:
	@echo "运行性能测试..."
	go run scripts/benchmark.go

# 运行优化版性能测试
benchmark-optimized:
	@echo "运行优化版性能测试..."
	go run qps_benchmark_optimized.go

# 运行超高性能版测试
benchmark-ultra:
	@echo "运行超高性能版测试..."
	go run qps_benchmark_ultra.go

# 运行极简版性能测试
benchmark-final:
	@echo "运行极简版性能测试..."
	go run qps_benchmark_final.go

# 运行所有版本性能测试对比
benchmark-all:
	@echo "运行所有版本性能测试对比..."
	./run_benchmark.sh

# 安装依赖
deps:
	@echo "安装依赖..."
	go mod tidy
	go mod download

# 格式化代码
fmt:
	@echo "格式化代码..."
	go fmt ./...

# 代码检查
lint:
	@echo "代码检查..."
	golangci-lint run

# 生成测试数据
generate-test-data:
	@echo "生成测试数据..."
	go run scripts/init_db.go

# 查看帮助
help:
	@echo "可用的命令:"
	@echo "  build              - 构建应用"
	@echo "  run                - 运行应用"
	@echo "  dev                - 开发模式运行"
	@echo "  test               - 运行测试"
	@echo "  clean              - 清理构建文件"
	@echo "  docker-build       - 构建Docker镜像"
	@echo "  docker-run         - 启动Docker服务"
	@echo "  docker-stop        - 停止Docker服务"
	@echo "  init-db            - 初始化数据库"
	@echo "  benchmark          - 运行性能测试"
	@echo "  benchmark-optimized - 运行优化版性能测试"
	@echo "  benchmark-ultra    - 运行超高性能版测试"
	@echo "  benchmark-final    - 运行极简版性能测试"
	@echo "  benchmark-all      - 运行所有版本性能测试对比"
	@echo "  deps               - 安装依赖"
	@echo "  fmt                - 格式化代码"
	@echo "  lint               - 代码检查"
	@echo "  generate-test-data - 生成测试数据"
	@echo "  help               - 显示帮助信息"

# 快速启动（包含数据库初始化）
quick-start: deps init-db dev 