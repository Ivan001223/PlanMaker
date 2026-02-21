.PHONY: help dev dev-infra dev-server dev-pdf stop build test clean

help: ## 显示帮助
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ==================== 开发环境 ====================
dev-infra: ## 启动基础设施 (PostgreSQL, Redis, MongoDB, RabbitMQ, MinIO)
	docker-compose up -d postgres redis mongodb rabbitmq minio

dev-server: ## 启动 Go 后端服务 (本地)
	cd server && go run main.go

dev-pdf: ## 启动 Python PDF 解析服务 (本地)
	cd pdf-service && uvicorn main:app --reload --host 0.0.0.0 --port 8000

dev: dev-infra ## 启动完整开发环境
	@echo "基础设施已启动，请手动运行 make dev-server 和 make dev-pdf"

stop: ## 停止所有服务
	docker-compose down

# ==================== 构建 ====================
build: ## 构建所有 Docker 镜像
	docker-compose build

build-server: ## 构建 Go 服务镜像
	docker-compose build server

build-pdf: ## 构建 PDF 服务镜像
	docker-compose build pdf-service

# ==================== 测试 ====================
test: test-server test-pdf ## 运行所有测试

test-server: ## 运行 Go 服务测试
	cd server && go test ./... -v -count=1

test-pdf: ## 运行 Python PDF 服务测试
	cd pdf-service && python -m pytest tests/ -v

# ==================== 数据库 ====================
db-reset: ## 重置数据库
	docker-compose down -v
	docker-compose up -d postgres redis mongodb
	@echo "等待数据库初始化..."
	sleep 10
	@echo "数据库已重置"

# ==================== 清理 ====================
clean: ## 清理所有容器和数据卷
	docker-compose down -v --rmi local
	@echo "已清理所有容器和数据卷"
