# Listen Stream 共享库 (Shared Packages)

本目录包含 Listen Stream 系统所有微服务共享的基础库。

## 📦 包列表

### 1. **config/** - 配置管理
配置分层策略实现，支持文件配置和 Consul KV 动态配置。

**核心功能**:
- ✅ 双层配置：基础设施（文件）+ 业务配置（Consul KV）
- ✅ 30秒本地缓存，减少 Consul 查询
- ✅ 配置变更监听（Watch 机制）
- ✅ 配置验证
- ✅ 热更新支持

**主要文件**:
- `types.go` - 配置结构定义
- `file.go` - 文件配置加载
- `consul.go` - Consul KV 配置服务
- `cache.go` - 本地缓存
- `watcher.go` - 配置变更监听
- `validator.go` - 配置验证
- `config.go` - 统一管理器

**使用示例**:
```go
// 创建配置管理器
mgr, err := config.NewManager(&config.ManagerConfig{
    ConfigFile:     "config/local.yaml",
    ConsulAddress:  "localhost:8500",
    ConsulKVPrefix: "listen-stream",
    WatchEnabled:   true,
})

// 获取配置
cfg := mgr.GetConfig()

// 监听配置变更
mgr.OnChange(func(cfg *config.BusinessConfig) error {
    log.Println("配置已更新")
    return nil
})
```

---

### 2. **db/** - 数据库封装
PostgreSQL 连接池管理，支持主从读写分离。

**核心功能**:
- ✅ 主从读写分离（1主多从）
- ✅ 连接池管理（pgBouncer 兼容）
- ✅ 健康检查
- ✅ 数据库迁移（golang-migrate）
- ✅ 连接统计

**主要文件**:
- `postgres.go` - PostgreSQL 连接池
- `health.go` - 健康检查
- `migration.go` - 数据库迁移工具

**使用示例**:
```go
// 创建数据库连接
db, err := db.NewPostgresDB(&db.PostgresConfig{
    Host:         "localhost",
    Port:         5432,
    User:         "postgres",
    Password:     "password",
    Database:     "listen_stream",
    MaxOpenConns: 25,
    MaxIdleConns: 5,
})

// 读操作使用副本
rows, err := db.QueryContext(ctx, "SELECT * FROM users")

// 写操作使用主库
result, err := db.ExecContext(ctx, "INSERT INTO users ...")
```

---

### 3. **redis/** - Redis 封装
Redis 客户端，支持单实例和集群模式。

**核心功能**:
- ✅ 单实例 / 集群模式自动切换
- ✅ Pub/Sub 消息订阅
- ✅ SingleFlight 缓存击穿保护
- ✅ 三级缓存（L1内存 + L2 Redis + L3降级）
- ✅ 键命名规范
- ✅ 连接池管理

**主要文件**:
- `client.go` - Redis 客户端
- `keys.go` - 键命名规范
- `pubsub.go` - Pub/Sub 封装
- `singleflight.go` - 缓存击穿保护

**使用示例**:
```go
// 创建 Redis 客户端
client, err := redis.NewClient(&redis.Config{
    Host:     "localhost",
    Port:     6379,
    PoolSize: 10,
})

// 基本操作
client.Set(ctx, "key", "value", time.Hour)
val, _ := client.Get(ctx, "key")

// SingleFlight 缓存
sfCache := redis.NewSingleFlightCache(client)
data, err := sfCache.Get(ctx, key, func() (interface{}, error) {
    return loadFromDB()
}, time.Hour)

// Pub/Sub
pubsub := redis.NewPubSub(client)
pubsub.Subscribe("channel")
pubsub.OnMessage("channel", func(ch, msg string) error {
    log.Println("收到消息:", msg)
    return nil
})
```

---

### 4. **crypto/** - 加密工具
提供 AES-256-GCM 加密、Argon2id 密码哈希等功能。

**核心功能**:
- ✅ AES-256-GCM 加密解密
- ✅ Argon2id 密码哈希
- ✅ 密钥生成
- ✅ 敏感数据脱敏

**主要文件**:
- `aes.go` - AES 加密
- `hash.go` - 密码哈希
- `keygen.go` - 密钥生成
- `mask.go` - 数据脱敏

---

### 5. **logger/** - 日志工具
基于 Zap 的结构化日志，支持日志轮转。

**核心功能**:
- ✅ 结构化 JSON 日志
- ✅ 日志轮转（按大小和时间）
- ✅ 多输出目标
- ✅ 敏感数据自动脱敏
- ✅ 高性能缓冲写入

**主要文件**:
- `logger.go` - Zap 封装
- `rotate.go` - 日志轮转

---

### 6. **grpc/** - gRPC 工具
gRPC 客户端/服务端封装和拦截器。

**核心功能**:
- ✅ 客户端连接池
- ✅ 服务端优雅关闭
- ✅ 拦截器（日志、追踪、认证、限流、恢复）
- ✅ 错误转换
- ✅ 服务发现集成

**主要文件**:
- `client.go` - gRPC 客户端
- `server.go` - gRPC 服务端
- `errors.go` - 错误转换
- `interceptor/` - 拦截器集合

---

### 7. **errors/** - 错误定义
统一的错误码和错误类型定义。

**核心功能**:
- ✅ 结构化错误（错误码 + HTTP 状态）
- ✅ 预定义常见错误
- ✅ 错误包装和链式调用
- ✅ 错误详情支持

**使用示例**:
```go
// 返回预定义错误
return errors.ErrUserNotFound

// 创建自定义错误
err := errors.New("CUSTOM_ERROR", "Something went wrong", http.StatusBadRequest)

// 包装错误
err := errors.Wrap(dbErr, "DATABASE_ERROR", "Failed to query", 500)

// 添加错误详情
err := errors.ErrValidationFailed.WithDetails(map[string]string{
    "field": "email",
    "reason": "invalid format",
})
```

---

### 8. **httputil/** - HTTP 工具
HTTP 响应格式化和中间件。

**核心功能**:
- ✅ 统一响应格式
- ✅ 分页响应
- ✅ Request ID 中间件
- ✅ CORS 中间件
- ✅ 安全头中间件

**使用示例**:
```go
// 成功响应
httputil.SuccessResponse(c, data)

// 错误响应
httputil.ErrorResponse(c, errors.ErrUserNotFound)

// 分页响应
httputil.PaginatedResponse(c, data, page, pageSize, totalItems)

// 中间件
router.Use(httputil.RequestIDMiddleware())
router.Use(httputil.CORSMiddleware())
router.Use(httputil.SecurityHeadersMiddleware())
```

---

### 9. **jwt/** - JWT 工具
JWT Token 生成和验证。

**核心功能**:
- ✅ Access Token 签发
- ✅ Refresh Token 签发
- ✅ Token 验证
- ✅ Token 版本控制
- ✅ IP 绑定支持

**使用示例**:
```go
// 创建 JWT 管理器
jwtMgr := jwt.NewManager(&jwt.Config{
    Secret:        "your-secret-key",
    Issuer:        "listen-stream",
    TokenExpiry:   time.Hour,
    RefreshExpiry: 7 * 24 * time.Hour,
})

// 生成 Token
token, err := jwtMgr.GenerateToken(userID, deviceID, tokenVersion, clientIP)

// 验证 Token
claims, err := jwtMgr.ValidateToken(token)

// 刷新 Token
newToken, err := jwtMgr.RefreshToken(refreshToken, newTokenVersion)
```

---

### 10. **breaker/** - 熔断器
实现断路器模式，防止级联故障。

**核心功能**:
- ✅ 三状态：关闭、打开、半开
- ✅ 失败阈值配置
- ✅ 自动恢复
- ✅ 统计信息

**使用示例**:
```go
// 创建熔断器
cb := breaker.New(&breaker.Config{
    Name:        "qq-music-api",
    MaxFailures: 5,
    Timeout:     30 * time.Second,
})

// 执行受保护的操作
err := cb.Execute(func() error {
    return callUpstreamAPI()
})

// 检查状态
state := cb.GetState()
```

---

### 11. **limiter/** - 速率限制
基于 Redis 的速率限制。

**核心功能**:
- ✅ 滑动窗口限流
- ✅ IP 限流
- ✅ 用户限流
- ✅ 自定义窗口大小

**使用示例**:
```go
// 创建速率限制器
limiter := limiter.NewRateLimiter(redisClient)

// 检查是否允许
allowed, err := limiter.Allow(ctx, "user:123", 100, time.Minute)

// IP 限流
ipLimiter := limiter.NewIPRateLimiter(redisClient, 1000, time.Hour)
allowed, _ := ipLimiter.Allow(ctx, "192.168.1.1")
```

---

### 12. **telemetry/** - 遥测工具
OpenTelemetry 集成（占位符实现）。

**核心功能**:
- 🚧 分布式追踪
- 🚧 指标收集
- 🚧 日志关联

> ⚠️ **注意**: 当前为简化实现，完整的 OpenTelemetry 集成将在后续版本中完成。

---

## 📚 依赖关系

```
config/
  ├─ spf13/viper
  └─ hashicorp/consul/api

db/
  ├─ lib/pq
  └─ golang-migrate/migrate

redis/
  ├─ redis/go-redis
  └─ golang.org/x/sync/singleflight

logger/
  └─ uber-go/zap

grpc/
  ├─ google.golang.org/grpc
  └─ google.golang.org/protobuf

jwt/
  └─ golang-jwt/jwt/v5

httputil/
  └─ gin-gonic/gin
```

## 🚀 快速开始

### 安装依赖

```bash
cd server/shared
go mod tidy
```

### 运行测试

```bash
# 测试所有包
go test ./pkg/...

# 测试特定包
go test ./pkg/config
go test ./pkg/logger -cover

# 运行基准测试
go test -bench=. ./pkg/redis
```

## 📖 设计原则

1. **高内聚低耦合**: 每个包独立，依赖最小化
2. **接口优先**: 核心功能定义接口，便于测试和扩展
3. **错误处理**: 统一错误定义，支持错误链
4. **可观测性**: 内置日志、指标、追踪支持
5. **性能优先**: 连接池、缓存、批处理优化
6. **安全第一**: 加密、认证、限流内置

## 🔧 配置示例

参见 `docs/config-example-local.yaml` 获取完整配置示例。

## 📝 开发指南

### 添加新包

1. 在 `pkg/` 下创建新目录
2. 编写核心功能代码
3. 添加单元测试（覆盖率 ≥ 80%）
4. 更新本 README
5. 添加使用示例

### 测试规范

- 单元测试文件: `*_test.go`
- 覆盖率要求: ≥ 80%
- 使用 `testify/assert` 断言
- 需要外部依赖时使用 `testcontainers`

## 📊 测试覆盖率

| 包 | 覆盖率 | 状态 |
|---|-------|------|
| crypto | 90%+ | ✅ |
| logger | 83.9% | ✅ |
| grpc | 80%+ | ✅ |
| config | 待测试 | 🚧 |
| db | 待测试 | 🚧 |
| redis | 待测试 | 🚧 |
| errors | N/A | ✅ |
| httputil | 待测试 | 🚧 |
| jwt | 待测试 | 🚧 |
| breaker | 待测试 | 🚧 |
| limiter | 待测试 | 🚧 |

## 🤝 贡献

请参考项目根目录的 CONTRIBUTING.md。

## 📄 许可证

本项目采用 MIT 许可证。
