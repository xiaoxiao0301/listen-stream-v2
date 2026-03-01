# Redis Pub/Sub 集成

## 概述

Step 27 实现了 Redis Pub/Sub，使 sync-svc 支持跨实例消息广播，实现水平扩展。

## 架构

```
Instance A                 Redis Pub/Sub                Instance B
   |                            |                            |
   |--- Publish sync:user:123 --->|                          |
   |                            |--- Broadcast ------------->|
   |                            |                            |---> User WebSocket
```

## 核心组件

### 1. Publisher (internal/pubsub/publisher.go)
- **PublishToUser**: 发布到特定用户频道 `sync:user:{userID}`
- **PublishBroadcast**: 发布到全局广播频道 `sync:broadcast`
- **BatchPublishToUsers**: 批量发布到多个用户
- **统计**: 发布计数、失败计数

### 2. Subscriber (internal/pubsub/subscriber.go)
- **订阅模式**: 使用 pattern matching (`sync:user:*`, `sync:broadcast`)
- **自动重连**: 连接断开时自动重连
- **实例过滤**: 自动丢弃来自自己实例的消息（防止循环）
- **统计**: 接收计数、处理计数、丢弃计数

### 3. Manager 集成 (internal/ws/manager.go)
- 启动时初始化 Publisher 和 Subscriber
- `handleBroadcast`: 广播消息时同时发布到 Redis
- `handlePubSubMessage`: 接收 Pub/Sub 消息并转发到本地 WebSocket 连接
- 关闭时停止 Subscriber 和 Publisher

## 频道命名规范

- **用户频道**: `sync:user:{userID}` - 发送给特定用户
- **广播频道**: `sync:broadcast` - 发送给所有在线用户
- **实例频道**: `sync:instance:{instanceID}` - 发送给特定实例（保留）

## 消息流程

### 场景: 用户在实例 A，事件从实例 B 发送

1. 实例 B 接收到同步事件（如 playlist.created）
2. 实例 B 的 Manager.Broadcast() 方法:
   - 检查用户是否在本地在线
   - 发布消息到 Redis: `PUBLISH sync:user:{userID} {message}`
3. Redis 将消息广播给所有订阅者
4. 实例 A 的 Subscriber 接收消息:
   - 检查 InstanceID（过滤自己的消息）
   - 调用 Manager.handlePubSubMessage()
5. 实例 A 的 Manager:
   - 检查用户是否在本地在线
   - 如果在线，广播到本地 WebSocket 连接

## 环境变量

```bash
# 实例标识（重要！多实例部署时必须唯一）
INSTANCE_ID=sync-svc-1

# Redis 配置
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=

# 其他配置
PORT=8004
JWT_SECRET=your-secret-key
```

## API 接口

### POST /api/v1/events
发布同步事件给单个用户

```json
{
  "user_id": "user-123",
  "type": "playlist.created",
  "data": {
    "playlist_id": "pl-456"
  }
}
```

### POST /api/v1/events/batch
批量发布事件给多个用户

```json
{
  "user_ids": ["user-1", "user-2", "user-3"],
  "type": "favorite.added",
  "data": {
    "item_id": "song-789"
  }
}
```

### POST /api/v1/events/broadcast
全局广播事件

```json
{
  "type": "system.announcement",
  "data": {
    "message": "System maintenance in 5 minutes"
  }
}
```

### GET /api/v1/stats/pubsub
获取 Pub/Sub 统计信息

响应:
```json
{
  "instance_id": "sync-svc-1",
  "publisher": {
    "total_published": 1234,
    "user_published": 1100,
    "broadcast_published": 134,
    "failed_published": 0
  },
  "subscriber": {
    "total_received": 2345,
    "user_received": 2200,
    "broadcast_received": 145,
    "processed_messages": 2300,
    "failed_messages": 0,
    "dropped_messages": 45,
    "reconnect_count": 0
  }
}
```

## 部署示例

### 单实例部署
```bash
export INSTANCE_ID=sync-svc-1
export REDIS_ADDR=localhost:6379
./sync-svc
```

### 多实例部署（负载均衡）
```bash
# 实例 1
export INSTANCE_ID=sync-svc-1
export PORT=8004
./sync-svc &

# 实例 2
export INSTANCE_ID=sync-svc-2
export PORT=8005
./sync-svc &

# 实例 3
export INSTANCE_ID=sync-svc-3
export PORT=8006
./sync-svc &

# Nginx 负载均衡
# upstream sync_svc {
#     server localhost:8004;
#     server localhost:8005;
#     server localhost:8006;
# }
```

## 测试

### 运行单元测试
```bash
# 测试 Pub/Sub 功能
go test -v ./internal/pubsub/...

# 测试所有组件
go test -v ./...
```

### 手动测试跨实例通信

#### 启动两个实例
```bash
# Terminal 1 - 实例 A
export INSTANCE_ID=instance-A
export PORT=8004
./sync-svc

# Terminal 2 - 实例 B
export INSTANCE_ID=instance-B
export PORT=8005
./sync-svc
```

#### 用户连接到实例 A
```bash
# WebSocket 连接（需要 JWT token）
wscat -c "ws://localhost:8004/ws?token=your-jwt-token"
```

#### 从实例 B 发送事件
```bash
curl -X POST http://localhost:8005/api/v1/events \
  -H "Authorization: Bearer your-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-123",
    "type": "playlist.created",
    "data": {"playlist_id": "pl-456"}
  }'
```

#### 验证
用户在实例 A 的 WebSocket 连接应该收到消息。

## 性能考虑

1. **消息大小**: 建议消息体 < 1KB，通过引用传递大数据
2. **发布频率**: Publisher 使用 Pipeline 批量操作，支持高吞吐
3. **订阅模式**: 使用 pattern matching 减少订阅数量
4. **实例过滤**: 自动过滤本实例消息，避免不必要的处理
5. **重连机制**: 自动重连，避免单点故障

## 监控指标

关键指标（通过 `/api/v1/stats/pubsub` 获取）:

- **Publisher**:
  - `failed_published`: 应为 0，非 0 表示 Redis 连接问题
  - `total_published`: 监控发布量
  
- **Subscriber**:
  - `dropped_messages`: 来自自己实例的消息数（正常）
  - `failed_messages`: 应为 0，非 0 表示处理错误
  - `reconnect_count`: 重连次数，频繁重连需要检查 Redis 稳定性

## 故障处理

### Redis 连接失败
- Subscriber 自动重连（默认 5 秒间隔，最多 10 次）
- Publisher 发布失败会记录 `failed_published`
- 离线消息队列确保消息不丢失

### 实例崩溃
- 其他实例继续服务
- 连接到崩溃实例的用户需要重新连接
- 离线消息队列保存未送达的消息

### 消息丢失
- 使用离线消息队列作为备份
- ACK 机制确认消息送达
- 重要消息可以通过 HTTP API 重试

## 未来优化

1. **消息持久化**: 关键事件持久化到数据库
2. **消息去重**: 基于 message ID 去重
3. **优先级队列**: 支持消息优先级
4. **消息追踪**: 端到端的消息追踪
5. **限流**: 按用户/频道限流
