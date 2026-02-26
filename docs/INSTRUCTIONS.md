# 🎵 Listen Stream 系统重构实施指南

> **版本**: 2.0  
> **日期**: 2026-02-26  
> **状态**: 准备开始 - 步骤0  
> **预计周期**: 14周

---

## 📖 目录

1. [项目概述](#1-项目概述)
2. [前置要求](#2-前置要求)
3. [环境准备](#3-环境准备)
4. [实施流程](#4-实施流程)
5. [开发规范](#5-开发规范)
6. [测试标准](#6-测试标准)
7. [部署流程](#7-部署流程)
8. [故障排查](#8-故障排查)

---

## 1. 项目概述

### 1.1 重构目标

基于 **p2.md** 全面分析报告，从零开始重构 Listen Stream 系统，实现：

**核心改进**:
- ✅ gRPC 内部通信（性能提升10x）
- ✅ 三级缓存架构（内存+Redis+降级）
- ✅ Consul 配置中心（热更新+版本控制）
- ✅ 多源 Fallback（QQ→Joox→NetEase→Kugou）
- ✅ WebSocket 心跳检测
- ✅ 分布式追踪（OpenTelemetry）
- ✅ Token 版本控制
- ✅ 多厂商 SMS Fallback
- ✅ 操作审计增强
- ✅ 实时指标统计

**架构特点**:
- **微服务架构**: 4个核心服务（auth-svc, proxy-svc, user-svc, sync-svc, admin-svc）
- **混合通信**: 客户端用HTTP REST，服务间用gRPC
- **服务治理**: Consul注册发现 + 熔断器 + 链路追踪
- **高可用**: 主从复制 + Redis集群 + 降级策略

### 1.2 技术栈总览

```yaml
后端:
  语言: Go 1.23.0
  Web框架: Gin 1.10.0
  RPC: gRPC 1.60.0 + Protobuf v3
  数据库: PostgreSQL 15 (主从) + Redis 7 (集群)
  服务治理: Consul 1.17
  追踪: OpenTelemetry + Jaeger
  监控: Prometheus + Grafana
  日志: Zap + ELK Stack

客户端:
  移动端: Flutter 3.22.0 + Riverpod
  管理端: React 19 + TypeScript + Zustand

基础设施:
  容器: Docker + Kubernetes
  CI/CD: GitHub Actions
  负载均衡: Nginx / K8s Ingress
```

---

## 2. 前置要求

### 2.1 开发工具

**必需**:
```bash
# Go环境
Go 1.23.0+                    # go version

# Protocol Buffers
protoc 24.0+                  # protoc --version
protoc-gen-go v1.31.0+        # go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
protoc-gen-go-grpc v1.3.0+    # go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 数据库工具
PostgreSQL 15+                # psql --version
Redis 7+                      # redis-cli --version

# 代码生成
sqlc 1.24.0+                  # sqlc version
golang-migrate 4.16.0+        # migrate -version

# 容器
Docker 24.0+                  # docker --version
Docker Compose 2.20.0+        # docker-compose --version
```

**可选**:
```bash
# 调试工具
grpcurl                       # gRPC命令行工具
Evans                         # gRPC交互式客户端
k6                            # 性能测试
golangci-lint                 # 代码检查

# 观测工具
Jaeger                        # 链路追踪UI
Grafana                       # 监控面板
```

### 2.2 技能要求

**后端开发者**:
- Go 语言熟练（goroutine、channel、context）
- gRPC / Protocol Buffers 基础
- PostgreSQL 查询优化
- Redis 使用（缓存、Pub/Sub）
- 分布式系统概念（CAP、BASE）

**前端开发者**:
- Flutter / Dart（移动端）
- React / TypeScript（管理端）
- WebSocket 编程
- 状态管理（Riverpod / Zustand）

**DevOps**:
- Docker / Kubernetes
- Consul 配置
- Prometheus / Grafana
- CI/CD 流水线

---

## 3. 环境准备

### 3.1 本地开发环境

#### Step 1: 克隆仓库

```bash
git clone https://github.com/your-org/listen-stream.git
cd listen-stream
```

#### Step 2: 启动依赖服务

```bash
# 启动PostgreSQL、Redis、Consul、Jaeger
docker-compose -f docker-compose.local.yml up -d

# 验证服务
docker ps                              # 查看容器状态
curl http://localhost:8500/ui/dc1      # Consul UI
curl http://localhost:16686            # Jaeger UI
psql -h localhost -U postgres -d listen_stream  # 测试PostgreSQL
redis-cli -h localhost ping            # 测试Redis
```

**docker-compose.local.yml 示例**:
```yaml
version: '3.9'

services:
  postgres:
    image: postgres:15-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres123
      POSTGRES_DB: listen_stream
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data

  consul:
    image: consul:1.17
    ports:
      - "8500:8500"
      - "8600:8600/udp"
    command: agent -server -bootstrap-expect=1 -ui -client=0.0.0.0

  jaeger:
    image: jaegertracing/all-in-one:1.52
    ports:
      - "16686:16686"   # UI
      - "14268:14268"   # Collector HTTP
      - "4318:4318"     # OTLP gRPC

volumes:
  postgres_data:
  redis_data:
```

#### Step 3: 初始化数据库

```bash
# 运行迁移脚本
migrate -path server/migrations -database "postgresql://postgres:postgres123@localhost:5432/listen_stream?sslmode=disable" up

# 验证表创建
psql -h localhost -U postgres -d listen_stream -c "\dt"
```

#### Step 4: 初始化Consul配置

```bash
# 运行配置初始化脚本
chmod +x docs/init-consul-config.sh
./docs/init-consul-config.sh

# 验证配置
consul kv get -recurse listen-stream/
```

#### Step 5: 安装Go依赖

```bash
cd server
go mod download
go mod verify
```

### 3.2 IDE配置

**VS Code 推荐插件**:
```json
{
  "recommendations": [
    "golang.go",                    // Go支持
    "zxh404.vscode-proto3",        // Protobuf语法
    "ms-azuretools.vscode-docker", // Docker支持
    "esbenp.prettier-vscode",      // 代码格式化
    "ms-kubernetes-tools.vscode-kubernetes-tools"
  ]
}
```

**GoLand 配置**:
- Enable Go Modules: `Preferences → Go → Go Modules → Enable`
- Protobuf Support: 安装 Protobuf 插件
- gRPC测试: 配置 gRPC 测试运行器

---

## 4. 实施流程

### 4.1 开发工作流

```
1. 创建功能分支
   └─> git checkout -b feature/step-XX-description

2. 实现功能
   ├─> 编写代码
   ├─> 编写单元测试
   ├─> 编写集成测试
   └─> 更新文档

3. 本地验证
   ├─> 运行测试: make test
   ├─> 代码检查: make lint
   ├─> 格式化: make fmt
   └─> 构建: make build

4. 提交代码
   ├─> git add .
   ├─> git commit -m "feat(step-XX): description"
   └─> git push origin feature/step-XX-description

5. 创建PR
   ├─> 填写PR模板
   ├─> 等待CI通过
   ├─> Code Review
   └─> 合并到main

6. 部署验证
   └─> 自动部署到Staging → 测试 → 生产
```

### 4.2 分支策略

```
main                    # 主分支（保护）
  ├─ feature/step-XX    # 功能分支
  ├─ bugfix/issue-XX    # 修复分支
  ├─ hotfix/critical    # 紧急修复
  └─ release/v1.0.0     # 发布分支
```

### 4.3 实施顺序（48步骤）

**阶段1: 基础设施（第1-2周）**:
```
步骤0: ★ Protobuf定义 + gRPC封装层（当前）
步骤1: crypto工具库
步骤2: 配置服务（Consul集成）
步骤3: 日志工具
步骤4: 数据库封装
步骤5: Redis封装
步骤6: 其他工具（错误、HTTP、JWT、追踪、熔断、限流）
```

**阶段2: 认证服务（第3-4周）**:
```
步骤7-14: auth-svc完整实现
  - 领域层 + 仓储层
  - SMS多厂商
  - JWT版本控制
  - 设备管理
  - gRPC服务 + HTTP处理层
  - 服务注册
```

**阶段3: API网关（第5周）**:
```
步骤15-19: proxy-svc完整实现
  - 三级缓存
  - 上游客户端（熔断+重试）
  - Fallback链（4源）
  - gRPC客户端 + 中间件栈
  - 路由配置
```

**阶段4: 用户内容服务（第6-7周）**:
```
步骤20-24: user-svc完整实现
  - 收藏、历史、歌单
  - gRPC服务 + HTTP处理层
  - Cron清理任务
```

**阶段5: 实时同步服务（第8周）**:
```
步骤25-28: sync-svc完整实现
  - WebSocket管理（心跳+限制）
  - 离线消息（队列+ACK）
  - Redis Pub/Sub集成
```

**阶段6: 管理服务（第9-10周）**:
```
步骤29-34: admin-svc完整实现
  - 管理员认证（2FA）
  - 配置管理（Consul双向同步）
  - 操作审计（告警）
  - 数据统计（聚合+实时）
  - 审计日志导出
```

**阶段7: 基础设施集成（第11周）**:
```
步骤35-38: 可观测性完善
  - Consul集群部署
  - OpenTelemetry全链路追踪
  - 监控告警（Prometheus+Grafana）
  - 日志聚合（ELK）
```

**阶段8: 客户端适配（第12周）**:
```
步骤39-40: 客户端对接新API
  - Flutter客户端适配
  - React管理后台适配
```

**阶段9: 测试与优化（第13周）**:
```
步骤41-44: 全面测试
  - 单元测试（目标80%覆盖率）
  - 集成测试
  - 性能测试（k6压测）
  - 混沌测试（故障注入）
```

**阶段10: 部署上线（第14周）**:
```
步骤45-48: 生产部署
  - Kubernetes部署
  - 生产环境准备
  - 灰度发布
  - 运维文档
```

### 4.4 每步骤标准流程

**开始步骤前**:
1. 阅读设计文档（listen-stream-redesign.md）
2. 理解功能需求和完成标准
3. 评估技术风险
4. 创建功能分支

**开发过程中**:
1. TDD：先写测试，再写实现
2. 遵循代码规范（见5.开发规范）
3. 持续提交（小步快跑）
4. 更新文档

**步骤完成后**:
1. ✅ 检查完成标准（见文档）
2. ✅ 运行所有测试（`make test`）
3. ✅ 代码覆盖率达标（`make cover`）
4. ✅ 性能满足预期（如有benchmark）
5. ✅ 创建PR并通过Review
6. ✅ 合并到主分支
7. ✅ 更新项目进度文档

---

## 5. 开发规范

### 5.1 代码组织

**目录结构**:
```
server/
├── shared/              # 共享库
│   ├── pkg/             # 公共包
│   │   ├── crypto/      # 加密工具
│   │   ├── config/      # 配置服务
│   │   ├── logger/      # 日志工具
│   │   ├── db/          # 数据库封装
│   │   ├── redis/       # Redis封装
│   │   ├── grpc/        # gRPC工具
│   │   └── ...
│   └── proto/           # Protobuf定义
│       ├── auth/v1/
│       ├── user/v1/
│       └── sync/v1/
│
├── services/            # 微服务
│   ├── auth-svc/
│   │   ├── cmd/         # 主程序入口
│   │   ├── internal/    # 内部实现
│   │   │   ├── domain/      # 领域模型
│   │   │   ├── repository/  # 数据访问
│   │   │   ├── service/     # 业务逻辑
│   │   │   ├── grpc/        # gRPC服务实现
│   │   │   ├── handler/     # HTTP处理器
│   │   │   └── middleware/  # 中间件
│   │   ├── config/      # 服务配置文件
│   │   └── Dockerfile
│   ├── proxy-svc/
│   ├── user-svc/
│   ├── sync-svc/
│   └── admin-svc/
│
├── migrations/          # 数据库迁移
├── scripts/             # 脚本工具
└── deployments/         # 部署配置
    ├── k8s/
    └── docker-compose/
```

### 5.2 命名规范

**Go代码**:
```go
// 包名：小写单词，无下划线
package userservice

// 接口：名词或形容词，以er结尾
type UserRepository interface {}
type Runnable interface {}

// 结构体：大驼峰
type UserService struct {}

// 方法：大驼峰（导出）或小驼峰（私有）
func (s *UserService) CreateUser() {}
func (s *UserService) validateEmail() {}

// 常量：大驼峰或全大写+下划线
const MaxRetries = 3
const DEFAULT_TIMEOUT = 30

// 变量：小驼峰
var userCache map[string]*User
```

**Protobuf**:
```protobuf
// 服务名：大驼峰
service AuthService {}

// RPC方法：大驼峰
rpc VerifyToken(VerifyTokenRequest) returns (VerifyTokenResponse);

// 消息类型：大驼峰
message VerifyTokenRequest {}

// 字段：snake_case
message User {
  string user_id = 1;
  string phone_number = 2;
  int64 created_at = 3;
}
```

**数据库**:
```sql
-- 表名：复数，snake_case
CREATE TABLE users (...);
CREATE TABLE device_sessions (...);

-- 字段名：snake_case
user_id, phone_number, created_at

-- 索引名：idx_{table}_{column}
CREATE INDEX idx_users_phone ON users(phone);

-- 外键名：fk_{table}_{ref_table}
CONSTRAINT fk_devices_users FOREIGN KEY (user_id) REFERENCES users(id)
```

### 5.3 注释规范

**包注释**:
```go
// Package crypto provides cryptographic utilities for the Listen Stream system.
//
// It includes AES-256-GCM encryption, Argon2id password hashing, and key generation tools.
//
// Example usage:
//
//	key, _ := crypto.GenerateAESKey()
//	ciphertext, _ := crypto.EncryptAES256GCM(plaintext, key)
package crypto
```

**函数注释**:
```go
// EncryptAES256GCM encrypts plaintext using AES-256-GCM mode.
//
// The key must be exactly 32 bytes (256 bits). The function generates a random
// 12-byte nonce for each encryption operation. The returned ciphertext format is:
// nonce(12 bytes) || ciphertext || authentication tag(16 bytes).
//
// Parameters:
//   - plaintext: The data to encrypt
//   - key: A 32-byte AES key
//
// Returns:
//   - []byte: The encrypted data with nonce and tag
//   - error: ErrInvalidKeySize if key is not 32 bytes, or encryption error
//
// Example:
//
//	key := make([]byte, 32)
//	rand.Read(key)
//	ciphertext, err := EncryptAES256GCM([]byte("secret"), key)
func EncryptAES256GCM(plaintext, key []byte) ([]byte, error)
```

**Protobuf注释**:
```protobuf
// AuthService provides authentication and token management for the Listen Stream platform.
//
// All RPCs require proper authentication except for the login flow.
service AuthService {
  // VerifyToken validates a JWT access token and returns the associated user information.
  //
  // This RPC is frequently called by proxy-svc for request authentication.
  // Performance target: P99 < 5ms
  rpc VerifyToken(VerifyTokenRequest) returns (VerifyTokenResponse);
}

// VerifyTokenRequest contains the token to be verified.
message VerifyTokenRequest {
  // The JWT access token in the format: "eyJhbGc..."
  string access_token = 1;
  
  // Optional client IP address for strict mode validation
  string client_ip = 2;
}
```

### 5.4 错误处理

**错误定义**:
```go
// 使用 errors.New 定义包级错误
var (
    ErrUserNotFound    = errors.New("user not found")
    ErrInvalidToken    = errors.New("invalid token")
    ErrDeviceLimitExceeded = errors.New("device limit exceeded")
)

// 使用 fmt.Errorf 包装错误（添加上下文）
if err != nil {
    return fmt.Errorf("failed to create user: %w", err)
}
```

**错误检查**:
```go
// 使用 errors.Is 检查错误类型
if errors.Is(err, ErrUserNotFound) {
    return http.StatusNotFound
}

// 使用 errors.As 提取错误
var validationErr *ValidationError
if errors.As(err, &validationErr) {
    return validationErr.Fields
}
```

**gRPC错误转换**:
```go
// HTTP错误 → gRPC Status
import "google.golang.org/grpc/status"
import "google.golang.org/grpc/codes"

func toGRPCError(err error) error {
    switch {
    case errors.Is(err, ErrUserNotFound):
        return status.Error(codes.NotFound, err.Error())
    case errors.Is(err, ErrInvalidToken):
        return status.Error(codes.Unauthenticated, err.Error())
    default:
        return status.Error(codes.Internal, "internal server error")
    }
}
```

### 5.5 日志规范

**日志级别**:
```go
// DEBUG: 详细调试信息（不应在生产启用）
logger.Debug("cache miss", zap.String("key", key))

// INFO: 一般信息（服务启动、配置加载）
logger.Info("service started", 
    zap.String("service", "auth-svc"),
    zap.Int("port", 8001))

// WARN: 警告信息（非致命错误、降级）
logger.Warn("upstream timeout, using fallback",
    zap.String("upstream", "qq_music"),
    zap.Duration("timeout", 5*time.Second))

// ERROR: 错误信息（业务错误）
logger.Error("failed to create user",
    zap.Error(err),
    zap.String("phone", phone))

// FATAL: 致命错误（服务无法启动）
logger.Fatal("failed to connect to database", zap.Error(err))
```

**结构化日志**:
```go
// ✅ 好的做法：结构化字段
logger.Info("user logged in",
    zap.String("user_id", userID),
    zap.String("device_id", deviceID),
    zap.String("ip", clientIP),
    zap.Duration("duration", time.Since(start)))

// ❌ 坏的做法：字符串拼接
logger.Info(fmt.Sprintf("user %s logged in from %s", userID, clientIP))
```

**敏感信息脱敏**:
```go
// 手机号、Token等敏感信息必须脱敏
logger.Info("SMS sent",
    zap.String("phone", crypto.MaskPhone(phone)),  // 138****5678
    zap.String("provider", "aliyun"))
```

### 5.6 测试规范

**单元测试**:
```go
// 文件命名: xxx_test.go
// 函数命名: Test{Function}_{Scenario}

func TestEncryptAES256GCM_ValidInput_Success(t *testing.T) {
    // Arrange（准备）
    key := make([]byte, 32)
    rand.Read(key)
    plaintext := []byte("secret message")
    
    // Act（执行）
    ciphertext, err := EncryptAES256GCM(plaintext, key)
    
    // Assert（断言）
    assert.NoError(t, err)
    assert.NotNil(t, ciphertext)
    assert.Greater(t, len(ciphertext), len(plaintext))
}

func TestEncryptAES256GCM_InvalidKey_ReturnsError(t *testing.T) {
    invalidKey := []byte("short")
    _, err := EncryptAES256GCM([]byte("data"), invalidKey)
    
    assert.Error(t, err)
    assert.Equal(t, ErrInvalidKeySize, err)
}
```

**表驱动测试**:
```go
func TestMaskPhone(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"valid phone", "13812345678", "138****5678"},
        {"short phone", "123", "123"},
        {"empty", "", ""},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := MaskPhone(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

**Mock使用**:
```go
//go:generate mockgen -source=user_repository.go -destination=mocks/user_repository_mock.go -package=mocks

func TestUserService_CreateUser(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    mockRepo := mocks.NewMockUserRepository(ctrl)
    mockRepo.EXPECT().
        Create(gomock.Any(), gomock.Any()).
        Return(nil)
    
    service := NewUserService(mockRepo)
    err := service.CreateUser(context.Background(), &User{})
    
    assert.NoError(t, err)
}
```

**集成测试**:
```go
// 使用 testcontainers-go 启动真实依赖
func TestIntegration_UserRepository(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // 启动PostgreSQL容器
    ctx := context.Background()
    postgresContainer, err := testcontainers.GenericContainer(ctx, 
        testcontainers.GenericContainerRequest{
            ContainerRequest: testcontainers.ContainerRequest{
                Image: "postgres:15-alpine",
                ExposedPorts: []string{"5432/tcp"},
                Env: map[string]string{
                    "POSTGRES_PASSWORD": "test",
                },
            },
            Started: true,
        })
    require.NoError(t, err)
    defer postgresContainer.Terminate(ctx)
    
    // 运行测试...
}
```

---

## 6. 测试标准

### 6.1 测试覆盖率要求

**最低标准**:
```
shared/pkg/*    ≥ 90%   # 共享库高标准
services/*      ≥ 80%   # 业务服务
cmd/*           ≥ 50%   # 主程序（主要测试配置加载）
```

**查看覆盖率**:
```bash
# 运行测试并生成覆盖率报告
make cover

# 或手动运行
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### 6.2 性能测试

**Benchmark编写**:
```go
func BenchmarkEncryptAES256GCM(b *testing.B) {
    key := make([]byte, 32)
    rand.Read(key)
    plaintext := make([]byte, 1024) // 1KB
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        EncryptAES256GCM(plaintext, key)
    }
}

// 运行: go test -bench=. -benchmem
```

**性能目标**:
```
加密（1KB）:       < 10 μs
密码哈希:          < 500 ms
gRPC调用:         < 5 ms (P99)
数据库查询:       < 10 ms (简单查询)
缓存读取:         < 1 ms
Token验证:        < 2 ms
```

### 6.3 压力测试

**k6脚本示例**:
```javascript
// test/load/auth_test.js
import http from 'k6/http';
import { check } from 'k6';

export let options = {
  stages: [
    { duration: '30s', target: 100 },   // 爬坡到100 VU
    { duration: '1m', target: 100 },    // 保持100 VU
    { duration: '30s', target: 0 },     // 降到0
  ],
  thresholds: {
    http_req_duration: ['p(99)<200'],   // 99%请求 < 200ms
    http_req_failed: ['rate<0.01'],     // 错误率 < 1%
  },
};

export default function () {
  let res = http.post('http://localhost:8001/api/v1/auth/verify-token', 
    JSON.stringify({ access_token: 'xxx' }), 
    { headers: { 'Content-Type': 'application/json' } }
  );
  
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 200ms': (r) => r.timings.duration < 200,
  });
}

// 运行: k6 run test/load/auth_test.js
```

---

## 7. 部署流程

### 7.1 本地开发环境

```bash
# 1. 启动依赖服务
docker-compose -f docker-compose.local.yml up -d

# 2. 运行数据库迁移
make migrate-up

# 3. 初始化Consul配置
./scripts/init-consul-config.sh

# 4. 启动服务（开发模式）
cd server/services/auth-svc
go run cmd/main.go

# 5. 验证服务
curl http://localhost:8001/health
grpcurl -plaintext localhost:9001 grpc.health.v1.Health/Check
```

### 7.2 Staging环境

```bash
# 1. 构建Docker镜像
make docker-build

# 2. 推送到镜像仓库
make docker-push

# 3. 部署到K8s
kubectl apply -f deployments/k8s/staging/

# 4. 验证部署
kubectl get pods -n listen-stream-staging
kubectl logs -f deployment/auth-svc -n listen-stream-staging

# 5. 运行冒烟测试
./scripts/smoke-test.sh staging
```

### 7.3 生产环境

```bash
# 1. 创建Release Tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 2. GitHub Actions自动构建和推送镜像

# 3. 灰度发布（10% → 50% → 100%）
kubectl patch deployment auth-svc -p '{"spec":{"replicas":2}}' -n listen-stream-prod
# 观察指标，逐步增加副本数

# 4. 健康检查
curl https://api.listenstream.com/health

# 5. 监控面板
# 打开Grafana: https://grafana.listenstream.com
# 检查错误率、延迟、QPS

# 6. 回滚准备
kubectl rollout history deployment/auth-svc -n listen-stream-prod
kubectl rollout undo deployment/auth-svc -n listen-stream-prod  # 如需回滚
```

---

## 8. 故障排查

### 8.1 常见问题

#### 问题1: gRPC连接失败

**症状**:
```
rpc error: code = Unavailable desc = connection error
```

**排查步骤**:
```bash
# 1. 检查服务是否启动
kubectl get pods -n listen-stream

# 2. 检查服务注册
curl http://localhost:8500/v1/catalog/service/auth-svc

# 3. 检查网络连通性
grpcurl -plaintext auth-svc.service.consul:9001 list

# 4. 查看服务日志
kubectl logs -f deployment/auth-svc --tail=100
```

**解决方案**:
- 确认服务已启动并注册到Consul
- 检查防火墙规则
- 验证gRPC端口配置

#### 问题2: 数据库连接池耗尽

**症状**:
```
pq: sorry, too many clients already
```

**排查步骤**:
```bash
# 1. 检查当前连接数
psql -h localhost -U postgres -c "SELECT count(*) FROM pg_stat_activity;"

# 2. 查看连接详情
psql -h localhost -U postgres -c "
  SELECT pid, usename, application_name, state, query_start 
  FROM pg_stat_activity 
  WHERE datname = 'listen_stream';
"

# 3. 检查配置
cat config/local.yaml | grep max_connections
```

**解决方案**:
- 增加`max_connections`配置
- 使用pgBouncer连接池
- 检查是否有连接泄漏（未关闭）

#### 问题3: Redis OOM

**症状**:
```
OOM command not allowed when used memory > 'maxmemory'
```

**排查步骤**:
```bash
# 1. 检查内存使用
redis-cli info memory

# 2. 查看最大键
redis-cli --bigkeys

# 3. 检查过期策略
redis-cli config get maxmemory-policy
```

**解决方案**:
- 增加Redis内存：`config set maxmemory 4gb`
- 设置淘汰策略：`config set maxmemory-policy allkeys-lru`
- 清理无用键
- 减少缓存TTL

#### 问题4: WebSocket连接中断

**症状**:
```
WebSocket connection closed unexpectedly
```

**排查步骤**:
```bash
# 1. 检查心跳日志
grep "heartbeat timeout" /var/log/sync-svc.log

# 2. 检查连接数
redis-cli get "ws:connection_count"

# 3. 查看Nginx超时配置
grep "proxy_read_timeout" /etc/nginx/nginx.conf
```

**解决方案**:
- 增加Nginx超时时间：`proxy_read_timeout 300s;`
- 客户端启用心跳
- 实现自动重连（指数退避）

### 8.2 日志查询

**查询特定RequestID的日志**:
```bash
# Elasticsearch
curl -X GET "localhost:9200/logs-*/_search" -H 'Content-Type: application/json' -d'
{
  "query": {
    "term": { "request_id": "req-12345" }
  }
}
'

# Kibana查询语法
request_id:"req-12345" AND level:"error"
```

**查询慢请求**:
```bash
# 查询响应时间 > 1s 的请求
duration:>1000 AND service:"proxy-svc"
```

### 8.3 性能分析

**CPU Profile**:
```bash
# 1. 启用pprof
# 代码中添加: import _ "net/http/pprof"

# 2. 收集30秒CPU数据
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof

# 3. 分析
go tool pprof -http=:8080 cpu.prof
```

**内存Profile**:
```bash
# 收集堆内存快照
curl http://localhost:6060/debug/pprof/heap > heap.prof

# 分析
go tool pprof -http=:8080 heap.prof
```

**Trace分析**:
```bash
# 收集trace数据
curl http://localhost:6060/debug/pprof/trace?seconds=5 > trace.out

# 查看
go tool trace trace.out
```

### 8.4 紧急恢复

**步骤1: 回滚代码**:
```bash
kubectl rollout undo deployment/auth-svc -n listen-stream-prod
```

**步骤2: 恢复配置**:
```bash
# 从Consul历史恢复
psql -h localhost -U postgres -d listen_stream -c "
  SELECT key, old_value 
  FROM config_history 
  WHERE version = (SELECT MAX(version) - 1 FROM config_history);
"

# 手动写回Consul
consul kv put listen-stream/common/jwt_secret "previous_value"
```

**步骤3: 恢复数据库**:
```bash
# 从备份恢复（最后手段）
pg_restore -h localhost -U postgres -d listen_stream backup.dump
```

---

## 9. 常用命令

### 9.1 Makefile目标

```makefile
# 开发
make run            # 启动当前服务
make test           # 运行测试
make cover          # 生成覆盖率报告
make lint           # 代码检查
make fmt            # 格式化代码

# Protobuf
make proto-gen      # 生成proto代码
make proto-clean    # 清理生成的代码

# 数据库
make migrate-up     # 运行迁移
make migrate-down   # 回滚迁移
make sqlc-gen       # 生成sqlc代码

# Docker
make docker-build   # 构建镜像
make docker-push    # 推送镜像
make docker-run     # 运行容器

# 部署
make deploy-staging # 部署到Staging
make deploy-prod    # 部署到生产

# 工具
make mock-gen       # 生成Mock
make check-deps     # 检查依赖更新
```

### 9.2 常用调试命令

```bash
# gRPC调试
grpcurl -plaintext localhost:9001 list                        # 列出服务
grpcurl -plaintext localhost:9001 list auth.v1.AuthService   # 列出方法
grpcurl -plaintext -d '{"access_token":"xxx"}' \
  localhost:9001 auth.v1.AuthService/VerifyToken             # 调用方法

# Consul操作
consul catalog services                                      # 列出服务
consul catalog nodes -service=auth-svc                       # 服务节点
consul kv get -recurse listen-stream/                        # 读取配置
consul kv put listen-stream/test "value"                     # 写入配置

# Redis操作
redis-cli keys "listen-stream:*"                             # 查看键
redis-cli ttl "cache:user:12345"                             # 查看过期时间
redis-cli monitor                                            # 监控命令
redis-cli --latency                                          # 延迟检测

# PostgreSQL操作
\l                                                           # 列出数据库
\dt                                                          # 列出表
\d users                                                     # 查看表结构
EXPLAIN ANALYZE SELECT ...                                   # 查询计划

# Kubernetes操作
kubectl get pods -n listen-stream                            # 查看Pod
kubectl describe pod auth-svc-xxx -n listen-stream           # Pod详情
kubectl logs -f auth-svc-xxx -n listen-stream                # 查看日志
kubectl exec -it auth-svc-xxx -n listen-stream -- /bin/sh   # 进入容器
kubectl port-forward svc/auth-svc 8001:8001                  # 端口转发
```

---

## 10. 资源链接

**文档**:
- [系统重构方案](docs/listen-stream-redesign.md)
- [Step 0: gRPC设计](docs/step0-grpc-design.md)
- [配置管理策略](docs/config-management-strategy.md)
- [覆盖率检查](docs/p2-coverage-check.md)

**工具文档**:
- [Go官方文档](https://go.dev/doc/)
- [gRPC Go教程](https://grpc.io/docs/languages/go/)
- [Protobuf指南](https://protobuf.dev/)
- [Consul文档](https://developer.hashicorp.com/consul)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)

**社区**:
- 技术讨论: Slack #listen-stream-dev
- 问题反馈: GitHub Issues
- 周会: 每周三 10:00 AM

---

## 11. 联系方式

**技术负责人**:
- 架构: @架构师Name
- 后端: @后端负责人
- 前端: @前端负责人
- DevOps: @运维负责人

**紧急联系**:
- On-call: +86-xxx-xxxx-xxxx
- PagerDuty: https://xxx.pagerduty.com

---

## 附录A: 术语表

| 术语 | 全称 | 说明 |
|-----|------|-----|
| gRPC | gRPC Remote Procedure Call | 高性能RPC框架 |
| Protobuf | Protocol Buffers | 数据序列化格式 |
| JWT | JSON Web Token | 无状态认证Token |
| 2FA | Two-Factor Authentication | 双因素认证 |
| TOTP | Time-based One-Time Password | 基于时间的一次性密码 |
| OTel | OpenTelemetry | 可观测性框架 |
| HPA | Horizontal Pod Autoscaler | K8s水平自动扩缩容 |
| WAF | Web Application Firewall | Web应用防火墙 |
| ELK | Elasticsearch, Logstash, Kibana | 日志分析栈 |
| KV | Key-Value | 键值存储 |

---

**版本历史**:
- v2.0 (2026-02-26): 完整实施指南，包含gRPC架构
- v1.0 (2026-02-20): 初始版本

**最后更新**: 2026-02-26

---

**📌 现在开始：回复 "继续" 开始实施步骤0（Protobuf定义 + gRPC封装）**
