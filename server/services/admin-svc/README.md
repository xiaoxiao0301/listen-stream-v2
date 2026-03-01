# admin-svc - 管理服务

Listen Stream 系统的管理服务，提供以下功能：

## 核心功能

### 1. 管理员认证
- ✅ 用户名/密码登录
- ✅ 双因素认证（TOTP）
- ✅ 角色权限管理（admin/operator/viewer）
- ✅ 登录日志记录

### 2. 配置管理
- ✅ Consul KV 配置存储
- ✅ 配置热更新（Redis Pub/Sub通知）
- ✅ 配置版本控制
- ✅ 配置变更历史（PostgreSQL）
- ✅ 30秒本地缓存

### 3. 操作审计
- ✅ 结构化操作日志（JSON详情）
- ✅ 异常活动检测
  - 批量禁用用户
  - 非工作时间敏感操作
  - 连续登录失败
  - 大量数据导出
- ✅ 自动告警（Redis Pub/Sub）

### 4. 数据统计
- ✅ 实时指标（Redis存储）
  - 在线用户数
  - 活跃会话数
  - 每分钟请求数
  - 每分钟错误数
  - 平均响应时间
- ✅ 每日统计（PostgreSQL聚合）
  - 用户统计（总数、新增、活跃）
  - 请求统计（总数、成功、失败、错误率）
  - 业务统计（收藏、歌单、播放次数）

### 5. 数据导出
- ✅ 审计日志导出（CSV/Excel）
- ✅ 统计数据导出（Excel）
- ✅ 支持日期范围筛选

## 技术栈

- **语言**: Go 1.23.0
- **Web框架**: Gin 1.10.0
- **数据库**: PostgreSQL 15
- **缓存**: Redis 7
- **服务注册**: Consul 1.17
- **TOTP**: pquerna/otp
- **Excel导出**: xuri/excelize

## 目录结构

```
admin-svc/
├── cmd/
│   └── main.go                   # 主程序入口
├── internal/
│   ├── domain/                   # 领域模型
│   │   ├── admin_user.go         # 管理员实体
│   │   ├── operation_log.go      # 操作日志实体
│   │   ├── daily_stats.go        # 每日统计实体
│   │   └── anomalous_activity.go # 异常活动实体
│   ├── repository/               # 数据访问层
│   │   └── queries/              # SQL查询文件（sqlc）
│   ├── service/                  # 服务层
│   │   ├── totp_service.go       # 双因素认证
│   │   ├── config_service.go     # 配置管理
│   │   ├── audit_service.go      # 操作审计
│   │   ├── stats_service.go      # 数据统计
│   │   └── export_service.go     # 数据导出
│   ├── handler/                  # HTTP处理层
│   │   ├── admin_handler.go      # 管理员API
│   │   ├── config_handler.go     # 配置API
│   │   ├── stats_handler.go      # 统计API
│   │   └── audit_handler.go      # 审计API
│   └── middleware/               # 中间件
│       └── middleware.go         # 认证、CORS等
├── migrations/                   # 数据库迁移
│   ├── 001_create_admin_tables.up.sql
│   └── 001_create_admin_tables.down.sql
├── sqlc.yaml                     # sqlc配置
├── go.mod
└── README.md
```

## 快速开始

### 1. 环境准备

```bash
# 启动依赖服务
docker-compose up -d redis consul postgresql
```

### 2. 数据库迁移

```bash
# 安装migrate工具
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 执行迁移
migrate -database "postgresql://postgres:password@localhost:5432/admin?sslmode=disable" \
        -path migrations up
```

### 3. 生成数据库代码

```bash
# 安装sqlc
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# 生成代码
sqlc generate
```

### 4. 启动服务

```bash
# 设置环境变量
export REDIS_ADDR=localhost:6379
export CONSUL_ADDR=localhost:8500
export HTTP_PORT=8005

# 运行服务
go run cmd/main.go
```

## API文档

### 管理员管理

#### 登录
```http
POST /api/v1/admins/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password",
  "totp_code": "123456"  // 可选，启用2FA时必须
}
```

#### 启用双因素认证
```http
POST /api/v1/admins/2fa/enable
Authorization: Bearer <token>

Response:
{
  "secret": "BASE32SECRET",
  "provisioning_uri": "otpauth://totp/...",
  "qr_code": "<base64 image>"
}
```

#### 验证并启用2FA
```http
POST /api/v1/admins/2fa/verify
Authorization: Bearer <token>
Content-Type: application/json

{
  "code": "123456"
}
```

### 配置管理

#### 获取配置
```http
GET /api/v1/configs/:key
Authorization: Bearer <token>
```

#### 更新配置
```http
PUT /api/v1/configs/:key
Authorization: Bearer <token>
Content-Type: application/json

{
  "value": "new-value",
  "reason": "更新原因"
}
```

#### 列出配置
```http
GET /api/v1/configs?prefix=common/
Authorization: Bearer <token>
```

#### 清除缓存
```http
POST /api/v1/configs/cache/clear
Authorization: Bearer <token>
```

### 统计数据

#### 获取实时统计
```http
GET /api/v1/stats/realtime
Authorization: Bearer <token>

Response:
{
  "online_users": 1234,
  "active_sessions": 567,
  "requests_per_min": 890,
  "errors_per_min": 12,
  "avg_response_time": 45,
  "timestamp": "2026-03-01T10:00:00Z"
}
```

#### 获取每日统计
```http
GET /api/v1/stats/daily?start_date=2026-03-01&end_date=2026-03-07
Authorization: Bearer <token>
```

#### 导出统计
```http
GET /api/v1/stats/daily/export?start_date=2026-03-01&end_date=2026-03-07&format=excel
Authorization: Bearer <token>
```

### 审计日志

#### 列出操作日志
```http
GET /api/v1/audit/logs?page=1&size=20&admin_id=xxx&operation=login
Authorization: Bearer <token>
```

#### 导出操作日志
```http
GET /api/v1/audit/logs/export?start_date=2026-03-01&end_date=2026-03-07&format=excel
Authorization: Bearer <token>
```

#### 列出异常活动
```http
GET /api/v1/audit/anomalies?page=1&size=20&severity=high&resolved=false
Authorization: Bearer <token>
```

#### 标记异常为已处理
```http
POST /api/v1/audit/anomalies/:id/resolve
Authorization: Bearer <token>
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `HTTP_PORT` | HTTP端口 | 8005 |
| `REDIS_ADDR` | Redis地址 | localhost:6379 |
| `REDIS_PASSWORD` | Redis密码 | (空) |
| `CONSUL_ADDR` | Consul地址 | localhost:8500 |
| `POSTGRES_DSN` | PostgreSQL连接串 | - |

## 配置结构（Consul KV）

```
listen-stream/
├── common/              # 共享配置
│   ├── jwt_secret       # JWT签名密钥
│   ├── jwt_version      # JWT版本号
│   └── aes_key          # AES加密密钥
├── api/                 # 第三方API
│   ├── qq_music/
│   │   ├── base_url
│   │   └── api_key
│   ├── joox/
│   └── netease/
├── sms/                 # 短信配置
│   ├── aliyun/
│   │   ├── access_key
│   │   ├── access_secret
│   │   └── sign_name
│   └── tencent/
└── features/            # 功能开关
    ├── token_ip_binding
    └── 2fa_required
```

## 数据库表

### admin_users
管理员用户表，包含：
- 基本信息（用户名、密码哈希、邮箱）
- 角色权限（admin/operator/viewer）
- 2FA设置（TOTP密钥、是否启用）
- 登录记录（最后登录时间、IP）

### operation_logs
操作日志表，记录所有管理操作：
- 操作详情（操作类型、资源、动作）
- 管理员信息（ID、姓名）
- 请求信息（IP、User Agent、Request ID）
- 执行结果（状态、错误信息、耗时）

### daily_stats
每日统计表，存储聚合数据：
- 用户统计（总数、新增、活跃）
- 请求统计（总数、成功、失败、错误率）
- 业务统计（收藏、歌单、播放）

### anomalous_activities
异常活动表，记录自动检测的异常：
- 异常类型和严重程度
- 触发管理员
- 处理状态（是否已处理、处理人、处理时间）

### config_histories
配置变更历史表：
- 配置键、旧值、新值
- 变更管理员、原因
- 版本号、是否可回滚

## 异常检测规则

| 异常类型 | 触发条件 | 严重程度 |
|----------|----------|----------|
| `bulk_disable` | 1小时内禁用≥20个用户 | High |
| `sensitive_op` | 非工作时间执行敏感操作 | Medium |
| `login_failure` | 10分钟内失败≥5次 | Medium |
| `data_leak` | 1小时内导出≥10次 | Critical |

工作时间定义：周一至周五 8:00-22:00

## 监控指标

### 实时指标（Redis）
- `stats:realtime:online_users` - 在线用户数
- `stats:realtime:active_sessions` - 活跃会话数
- `stats:realtime:requests_per_min` - 每分钟请求数
- `stats:realtime:errors_per_min` - 每分钟错误数
- `stats:realtime:avg_response_time` - 平均响应时间（ms）

### 每日指标（Redis + PostgreSQL）
- `daily:{date}:total_users` - 总用户数
- `daily:{date}:new_users` - 新增用户数
- `daily:{date}:active_users` - 活跃用户数
- `daily:{date}:total_requests` - 总请求数
- ...（其他指标）

## 告警通道

异常活动告警通过 Redis Pub/Sub 发布到 `admin:alerts` 频道。

可以订阅该频道并转发到：
- Slack Webhook
- Email
- 钉钉机器人
- PagerDuty

## 开发指南

### 添加新的配置项

1. 在Consul KV中创建配置：
```bash
consul kv put listen-stream/my-config/key "value"
```

2. 使用ConfigService读取：
```go
value, err := configSvc.Get(ctx, "my-config/key")
```

### 添加新的异常检测规则

编辑 `internal/service/audit_service.go` 的 `CheckAnomalousActivity` 方法：

```go
// 检查自定义异常
if log.Operation == "custom_op" {
    count, err := s.countRecentOperations(ctx, log.AdminID, "custom_op", 1*time.Hour)
    if err == nil && count >= 10 {
        return s.createAnomaly(
            "custom_anomaly",
            domain.SeverityHigh,
            "自定义异常描述",
            log.AdminID,
            log.AdminName,
        ), nil
    }
}
```

### 添加新的统计指标

1. 在Redis中存储实时数据：
```go
statsSvc.IncrementDailyCounter(ctx, time.Now(), "my_metric")
```

2. 在`daily_stats`表添加新列（迁移文件）

3. 更新聚合逻辑（`stats_service.go`）

## 待完成功能（TODO）

- [ ] 管理员密码修改
- [ ] 管理员权限细粒度控制
- [ ] 配置回滚功能
- [ ] 审计日志全文搜索
- [ ] 实时WebSocket推送异常告警
- [ ] 自定义告警规则

## License

MIT
