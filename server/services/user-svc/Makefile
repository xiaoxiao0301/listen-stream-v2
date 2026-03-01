.PHONY: help build run test clean migrate-up migrate-down sqlc

help: ## 显示帮助信息
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## 编译服务
	go build -o bin/user-svc cmd/main.go

run: ## 运行服务
	go run cmd/main.go

test: ## 运行测试
	go test -v ./...

clean: ## 清理编译产物
	rm -rf bin/

sqlc: ## 生成sqlc代码
	sqlc generate

migrate-up: ## 运行数据库迁移（升级）
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down: ## 回滚数据库迁移
	migrate -path migrations -database "$(DATABASE_URL)" down

migrate-create: ## 创建新的迁移文件
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

docker-build: ## 构建Docker镜像
	docker build -t user-svc:latest .

docker-run: ## 运行Docker容器
	docker run -p 8003:8003 -p 9003:9003 user-svc:latest

lint: ## 代码检查
	golangci-lint run

fmt: ## 格式化代码
	go fmt ./...

tidy: ## 整理依赖
	go mod tidy

proto: ## 生成proto代码
	cd ../../shared && make proto
