# 配置管理分层策略

## 📐 架构决策

### 配置分类

#### 1️⃣ 基础设施配置（配置文件/环境变量）
**特点**: 启动时必需、很少变更、环境相关

```yaml
# config/local.yaml
infrastructure:
  postgres:
    host: localhost
    port: 5432
    database: listen_stream
    max_connections: 100
    ssl_mode: disable
    
  redis:
    addresses:
      - localhost:6379
      - localhost:6380
      - localhost:6381
    password: ""
    db: 0
    pool_size: 50
    
  service:
    name: auth-svc
    http_port: 8001
    grpc_port: 9001
    environment: local  # local | staging | production
```

**读取方式**:
```go
// 启动时一次性读取，不支持热更新
cfg, err := config.LoadFromFile("config/local.yaml")
// 或从环境变量
cfg, err := config.LoadFromEnv()
```

---

#### 2️⃣ 业务配置（Consul KV统一管理）
**特点**: 需要热更新、跨服务共享、敏感信息

```
consul kv结构:
├── listen-stream/
│   ├── common/              # 所有服务共享
│   │   ├── jwt_secret       # JWT签名密钥
│   │   ├── jwt_version      # JWT密钥版本号
│   │   └── aes_key          # 配置加密密钥
│   │
│   ├── api/                 # 第三方API配置
│   │   ├── qq_music/
│   │   │   ├── base_url
│   │   │   ├── api_key
│   │   │   ├── cookie
│   │   │   └── enabled
│   │   ├── joox/
│   │   │   ├── base_url
│   │   │   └── enabled
│   │   └── netease/...
│   │
│   ├── sms/                 # 短信配置
│   │   ├── aliyun/
│   │   │   ├── access_key
│   │   │   ├── secret_key
│   │   │   ├── sign_name
│   │   │   └── template_code
│   │   ├── tencent/...
│   │   └── provider_priority  # ["aliyun", "tencent", "twilio"]
│   │
│   └── features/            # 功能开关
│       ├── token_ip_binding    # true/false
│       ├── device_fingerprint  # true/false
│       └── strict_mode         # true/false
```

**读取方式**:
```go
// 启动时读取 + Watch变更
configSvc := consul.NewConfigService(consulClient)

// 读取配置（带30秒本地缓存）
jwtSecret, err := configSvc.GetString("listen-stream/common/jwt_secret")

// 监听配置变更
configSvc.Watch("listen-stream/api/qq_music/base_url", func(newValue string) {
    // 配置变更时自动回调
    upstreamClient.UpdateBaseURL(newValue)
})
```

---

## 🏗️ 配置服务架构

```
┌─────────────────────────────────────────────────────────┐
│                   应用服务                               │
│              (auth-svc / proxy-svc / ...)               │
│                                                          │
│  启动阶段:                                               │
│  1. 读取 config/local.yaml (PostgreSQL/Redis)          │
│  2. 连接 Consul                                         │
│  3. 读取业务配置到本地缓存(30s TTL)                     │
│  4. 启动 Watch 监听配置变更                              │
│                                                          │
│  运行阶段:                                               │
│  - 读取配置: 优先本地缓存(30s内)                        │
│  - 缓存过期: 自动从Consul拉取最新值                     │
│  - 配置变更: Watch回调立即生效                          │
└─────────────────┬───────────────────────────────────────┘
                  │
                  │ 读取业务配置
                  ▼
┌─────────────────────────────────────────────────────────┐
│               Consul KV (配置中心)                       │
│  • 存储业务配置                                          │
│  • 版本控制                                              │
│  • Watch机制                                            │
│  • 高可用(3节点集群)                                     │
└─────────────────┬───────────────────────────────────────┘
                  │
                  │ 管理员修改配置
                  ▼
┌─────────────────────────────────────────────────────────┐
│               admin-svc (管理后台)                       │
│  PUT /admin/config/api/qq-music                         │
│  {                                                       │
│    "base_url": "https://new-api.qq.com",               │
│    "enabled": true                                      │
│  }                                                       │
│                                                          │
│  1. 验证配置                                             │
│  2. 写入 Consul KV                                      │
│  3. 写入 PostgreSQL (history)                           │
│  4. 发布 Redis Pub/Sub 通知                             │
└─────────────────────────────────────────────────────────┘
                  │
                  │ Redis Pub/Sub "config:changed"
                  ▼
┌─────────────────────────────────────────────────────────┐
│              所有服务实例                                 │
│  接收通知 → 清除本地缓存 → 立即生效                      │
└─────────────────────────────────────────────────────────┘
```

---

## 📝 配置管理库实现

### 目录结构
```
server/shared/pkg/config/
├── file.go         # 文件配置加载器
├── consul.go       # Consul KV配置服务
├── cache.go        # 本地缓存(30s TTL)
├── watcher.go      # 配置变更监听
├── types.go        # 配置结构定义
└── validator.go    # 配置验证
```

### 核心接口

```go
package config

import (
    "context"
    "time"
)

// ===== 文件配置 =====

// InfraConfig 基础设施配置(启动时加载)
type InfraConfig struct {
    Postgres PostgresConfig
    Redis    RedisConfig
    Service  ServiceConfig
}

type PostgresConfig struct {
    Host           string
    Port           int
    Database       string
    User           string
    Password       string
    MaxConnections int
    SSLMode        string
}

type RedisConfig struct {
    Addresses  []string
    Password   string
    DB         int
    PoolSize   int
}

type ServiceConfig struct {
    Name        string
    HTTPPort    int
    GRPCPort    int
    Environment string // local | staging | production
}

// LoadFromFile 从YAML文件加载配置
func LoadFromFile(path string) (*InfraConfig, error)

// LoadFromEnv 从环境变量加载配置
func LoadFromEnv() (*InfraConfig, error)

// ===== Consul配置服务 =====

// ConsulConfigService 统一配置服务
type ConsulConfigService interface {
    // 读取配置(带缓存)
    GetString(ctx context.Context, key string) (string, error)
    GetInt(ctx context.Context, key string) (int, error)
    GetBool(ctx context.Context, key string) (bool, error)
    GetJSON(ctx context.Context, key string, v interface{}) error
    
    // 写入配置(管理员操作)
    SetString(ctx context.Context, key, value string) error
    SetJSON(ctx context.Context, key string, v interface{}) error
    
    // 删除配置
    Delete(ctx context.Context, key string) error
    
    // 监听配置变更
    Watch(ctx context.Context, key string, callback func(newValue string)) error
    
    // 列出所有配置
    ListKeys(ctx context.Context, prefix string) ([]string, error)
    
    // 清除本地缓存
    InvalidateCache(key string)
}

// NewConsulConfigService 创建Consul配置服务
func NewConsulConfigService(consulAddr string, opts ...Option) (ConsulConfigService, error)
```

### 使用示例

#### 1. auth-svc 启动流程

```go
// cmd/main.go
package main

import (
    "context"
    "log"
    
    "yourorg/listen-stream/shared/pkg/config"
)

func main() {
    ctx := context.Background()
    
    // 1. 加载基础设施配置(PostgreSQL/Redis)
    infraCfg, err := config.LoadFromFile("config/local.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    // 2. 连接数据库
    db, err := connectPostgres(infraCfg.Postgres)
    if err != nil {
        log.Fatal(err)
    }
    
    // 3. 连接Redis
    rdb, err := connectRedis(infraCfg.Redis)
    if err != nil {
        log.Fatal(err)
    }
    
    // 4. 初始化Consul配置服务
    consulCfg, err := config.NewConsulConfigService(
        "localhost:8500",
        config.WithCacheTTL(30 * time.Second),
        config.WithNamespace("listen-stream"),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // 5. 读取业务配置
    jwtSecret, err := consulCfg.GetString(ctx, "common/jwt_secret")
    if err != nil {
        log.Fatal(err)
    }
    
    // 6. 监听配置变更
    consulCfg.Watch(ctx, "common/jwt_version", func(newVersion string) {
        log.Printf("JWT version changed to: %s", newVersion)
        // 清除Token缓存
        tokenCache.Clear()
    })
    
    // 7. 启动HTTP和gRPC服务器
    go startHTTPServer(infraCfg.Service.HTTPPort, db, rdb, consulCfg)
    go startGRPCServer(infraCfg.Service.GRPCPort, db, rdb, consulCfg)
    
    // 8. 优雅关闭
    <-ctx.Done()
}
```

#### 2. proxy-svc 读取上游API配置

```go
// internal/upstream/qq_music.go
package upstream

import (
    "context"
    "yourorg/listen-stream/shared/pkg/config"
)

type QQMusicClient struct {
    configSvc config.ConsulConfigService
    baseURL   string
    apiKey    string
}

func NewQQMusicClient(configSvc config.ConsulConfigService) *QQMusicClient {
    ctx := context.Background()
    
    // 读取配置
    baseURL, _ := configSvc.GetString(ctx, "api/qq_music/base_url")
    apiKey, _ := configSvc.GetString(ctx, "api/qq_music/api_key")
    
    client := &QQMusicClient{
        configSvc: configSvc,
        baseURL:   baseURL,
        apiKey:    apiKey,
    }
    
    // 监听配置变更
    configSvc.Watch(ctx, "api/qq_music/base_url", func(newURL string) {
        client.baseURL = newURL
        log.Printf("QQ Music base URL updated to: %s", newURL)
    })
    
    return client
}
```

#### 3. admin-svc 修改配置

```go
// internal/handler/config_handler.go
package handler

func (h *ConfigHandler) UpdateAPIConfig(c *gin.Context) {
    var req struct {
        BaseURL string `json:"base_url"`
        APIKey  string `json:"api_key"`
        Enabled bool   `json:"enabled"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    ctx := c.Request.Context()
    
    // 1. 写入Consul KV
    if err := h.consulCfg.SetString(ctx, "api/qq_music/base_url", req.BaseURL); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // 2. 写入PostgreSQL历史记录
    h.repo.SaveConfigHistory(ctx, ConfigHistory{
        Key:       "api/qq_music/base_url",
        OldValue:  oldValue,
        NewValue:  req.BaseURL,
        ChangedBy: c.GetString("admin_id"),
        ChangedAt: time.Now(),
    })
    
    // 3. 发布Redis Pub/Sub通知(所有实例立即清除缓存)
    h.rdb.Publish(ctx, "config:changed", "api/qq_music/base_url")
    
    c.JSON(200, gin.H{"success": true})
}
```

---

## 🔒 敏感配置加密

### Consul中的敏感值加密存储

```go
// 写入时加密
func (s *ConsulConfigService) SetSecretString(ctx context.Context, key, value string) error {
    // 使用AES-256-GCM加密
    encrypted, err := s.crypto.Encrypt([]byte(value))
    if err != nil {
        return err
    }
    
    // 存储到Consul(Base64编码)
    return s.kv.Put(&api.KVPair{
        Key:   key,
        Value: encrypted,
        Flags: 1, // 标记为加密值
    }, nil)
}

// 读取时解密
func (s *ConsulConfigService) GetSecretString(ctx context.Context, key string) (string, error) {
    pair, _, err := s.kv.Get(key, nil)
    if err != nil {
        return "", err
    }
    
    if pair.Flags == 1 { // 加密值
        decrypted, err := s.crypto.Decrypt(pair.Value)
        if err != nil {
            return "", err
        }
        return string(decrypted), nil
    }
    
    return string(pair.Value), nil
}
```

---

## ✅ 配置管理最佳实践

### 1. 配置分层原则
- **基础设施配置**: 文件/环境变量（PostgreSQL、Redis）
- **业务配置**: Consul KV（JWT密钥、API密钥、功能开关）
- **运行时配置**: 动态调整（限流阈值、超时时间）

### 2. 配置变更流程
```
管理员修改配置
    ↓
admin-svc验证
    ↓
写入Consul KV + PostgreSQL历史
    ↓
发布Redis Pub/Sub通知
    ↓
所有服务实例清除缓存
    ↓
下次读取时自动拉取新值
```

### 3. 配置热更新策略
- **立即生效**: API密钥、功能开关（清除缓存）
- **延迟生效**: JWT密钥（新旧并存，version字段控制）
- **重启生效**: PostgreSQL连接池大小（需要重启）

### 4. 容错机制
- Consul不可用 → 使用本地缓存（stale值）
- 配置格式错误 → 使用默认值 + 告警
- 加密密钥丢失 → 无法启动（fail-fast）

---

## 📊 配置优先级

当同一配置在多处定义时，优先级从高到低：

1. **Consul KV** (最高优先级，支持热更新)
2. **环境变量** (容器化部署常用)
3. **配置文件** (config/local.yaml)
4. **默认值** (代码中硬编码，最低优先级)

---

## 🎯 步骤0调整

**在Protobuf定义中，配置相关的消息不需要定义**，因为配置通过Consul KV + 文件管理，不通过gRPC传递。

**步骤2（配置服务）提前到步骤1之后**，因为后续步骤都依赖配置服务。

**调整后的顺序**:
- 步骤0: Protobuf + gRPC封装
- 步骤1: crypto工具库
- 步骤2: **配置服务（重要性提升）**
- 步骤3: 日志工具
- ...

---

**完成时间估计**: 配置管理策略设计已完成，实现在步骤2中进行。
