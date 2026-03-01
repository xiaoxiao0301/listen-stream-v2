# sync-svc API 文档

## 概述

sync-svc 是实时同步服务，提供WebSocket连接和HTTP事件推送接口。

**基础URL**: `http://localhost:8004`  
**WebSocket URL**: `ws://localhost:8004/ws`  
**健康检查**: `GET /health`

---

## 认证

大多数API需要JWT认证。在请求头中包含：

```http
Authorization: Bearer <your-jwt-token>
```

---

## WebSocket API

### 1. 建立WebSocket连接

**端点**: `GET /ws`  
**认证**: 必需（JWT）  
**协议**: WebSocket

#### 连接URL示例

```
ws://localhost:8004/ws?token=<your-jwt-token>
```

或在请求头中传递：

```http
GET /ws HTTP/1.1
Host: localhost:8004
Upgrade: websocket
Connection: Upgrade
Authorization: Bearer <your-jwt-token>
```

#### 连接成功

服务器会推送离线消息（如果有），客户端开始接收实时事件。

#### 心跳机制

**服务器Ping**: 每30秒发送一次Ping  
**客户端响应**: 必须在60秒内响应Pong  
**超时**: 60秒无Pong则断开连接

#### 客户端发送Ping

```json
{
  "type": "ping",
  "timestamp": "2026-03-01T12:00:00Z"
}
```

**服务器响应**:
```json
{
  "type": "pong",
  "timestamp": "2026-03-01T12:00:00Z"
}
```

#### 接收同步事件

**收藏添加事件**:
```json
{
  "id": "msg-123",
  "type": "favorite.added",
  "user_id": "user-456",
  "data": {
    "item_id": "song-789",
    "item_type": "song",
    "added_at": "2026-03-01T12:00:00Z"
  },
  "timestamp": "2026-03-01T12:00:00Z"
}
```

**歌单创建事件**:
```json
{
  "id": "msg-124",
  "type": "playlist.created",
  "user_id": "user-456",
  "data": {
    "playlist_id": "pl-101",
    "name": "My Favorites",
    "created_at": "2026-03-01T12:00:00Z"
  },
  "timestamp": "2026-03-01T12:00:00Z"
}
```

#### 错误消息

```json
{
  "type": "error",
  "code": "invalid_message_format",
  "message": "Message must be valid JSON",
  "timestamp": "2026-03-01T12:00:00Z"
}
```

#### 重连策略

建议使用指数退避：
- 首次重连: 1秒后
- 第二次: 2秒后
- 第三次: 4秒后
- ...
- 最大间隔: 30秒

---

## HTTP API

### 事件推送 API

#### 1. 发布单用户事件

**端点**: `POST /api/v1/events`  
**认证**: 必需（JWT）  
**限流**: 100 req/min per client

**请求体**:
```json
{
  "user_id": "user-123",
  "type": "favorite.added",
  "data": {
    "item_id": "song-456",
    "item_type": "song"
  }
}
```

**响应** (200):
```json
{
  "success": true,
  "message_id": "msg-789",
  "user_id": "user-123",
  "type": "favorite.added",
  "timestamp": "2026-03-01T12:00:00Z"
}
```

**错误响应** (400):
```json
{
  "error": "invalid_message_type",
  "message": "unsupported message type"
}
```

**支持的消息类型**:
- `favorite.added`: 收藏添加
- `favorite.removed`: 收藏移除
- `playlist.created`: 歌单创建
- `playlist.updated`: 歌单更新
- `playlist.deleted`: 歌单删除
- `playlist.song.added`: 歌单歌曲添加
- `playlist.song.removed`: 歌单歌曲移除
- `history.added`: 播放历史添加

---

#### 2. 批量发布事件

**端点**: `POST /api/v1/events/batch`  
**认证**: 必需（JWT）

**请求体**:
```json
{
  "user_ids": ["user-1", "user-2", "user-3"],
  "type": "favorite.added",
  "data": {
    "item_id": "song-456"
  }
}
```

**响应** (200):
```json
{
  "success": true,
  "message_id": "msg-789",
  "user_count": 3,
  "type": "favorite.added",
  "timestamp": "2026-03-01T12:00:00Z"
}
```

---

#### 3. 全局广播

**端点**: `POST /api/v1/events/broadcast`  
**认证**: 必需（JWT）

**请求体**:
```json
{
  "type": "system.announcement",
  "data": {
    "message": "System maintenance in 5 minutes",
    "level": "warning"
  }
}
```

**响应** (200):
```json
{
  "success": true,
  "message_id": "msg-790",
  "type": "system.announcement",
  "timestamp": "2026-03-01T12:00:00Z",
  "online_users": 1234
}
```

---

### 离线消息 API

#### 1. 拉取离线消息

**端点**: `GET /api/v1/offline/messages`  
**认证**: 必需（JWT）

**查询参数**:
- `limit` (可选): 最多拉取条数，默认50

**请求示例**:
```
GET /api/v1/offline/messages?limit=20
```

**响应** (200):
```json
{
  "messages": [
    {
      "id": "msg-001",
      "user_id": "user-123",
      "type": "favorite.added",
      "data": {
        "item_id": "song-456"
      },
      "ack_token": "ack-abc123",
      "created_at": "2026-03-01T11:50:00Z",
      "expires_at": "2026-03-08T11:50:00Z"
    }
  ],
  "count": 1
}
```

---

#### 2. 确认单条消息

**端点**: `POST /api/v1/offline/ack`  
**认证**: 必需（JWT）

**请求体**:
```json
{
  "message_id": "msg-001",
  "ack_token": "ack-abc123"
}
```

**响应** (200):
```json
{
  "message": "acknowledged"
}
```

**错误响应** (400):
```json
{
  "error": "message not found or already acknowledged"
}
```

---

#### 3. 批量确认消息

**端点**: `POST /api/v1/offline/ack/batch`  
**认证**: 必需（JWT）

**请求体**:
```json
{
  "acks": [
    {
      "message_id": "msg-001",
      "ack_token": "ack-abc123"
    },
    {
      "message_id": "msg-002",
      "ack_token": "ack-def456"
    }
  ]
}
```

**响应** (200):
```json
{
  "message": "acknowledged",
  "count": 2
}
```

---

#### 4. 获取离线消息数量

**端点**: `GET /api/v1/offline/count`  
**认证**: 必需（JWT）

**响应** (200):
```json
{
  "user_id": "user-123",
  "count": 5
}
```

---

### 统计 API

#### 1. 获取系统统计

**端点**: `GET /api/v1/stats`  
**认证**: 可选

**响应** (200):
```json
{
  "total_registered": 10000,
  "total_unregistered": 9500,
  "current_connections": 500,
  "max_connections": 10000,
  "available_connections": 9500,
  "instance_id": "sync-svc-1",
  "rooms": {
    "total": 450,
    "total_connections": 500,
    "max_connections_per_user": 3,
    "avg_connections_per_user": 1.1
  },
  "heartbeat": {
    "total": 500,
    "healthy": 480,
    "warning": 15,
    "unhealthy": 5,
    "max_delay": 2.5,
    "min_delay": 0.1,
    "avg_delay": 0.5
  },
  "pubsub": {
    "publisher": {
      "total_published": 5000,
      "user_published": 4800,
      "broadcast_published": 200,
      "failed_published": 0
    },
    "subscriber": {
      "total_received": 3000,
      "user_received": 2900,
      "broadcast_received": 100,
      "processed_messages": 2950,
      "failed_messages": 50,
      "dropped_messages": 100,
      "reconnect_count": 0
    }
  }
}
```

---

#### 2. 获取在线用户

**端点**: `GET /api/v1/online-users`  
**认证**: 可选

**响应** (200):
```json
{
  "online_users": ["user-1", "user-2", "user-3"],
  "count": 3
}
```

---

#### 3. 检查用户在线状态

**端点**: `GET /api/v1/users/:user_id/online`  
**认证**: 可选

**响应** (200):
```json
{
  "user_id": "user-123",
  "online": true,
  "connection_count": 2
}
```

---

#### 4. 获取离线消息统计

**端点**: `GET /api/v1/offline/stats`  
**认证**: 可选

**响应** (200):
```json
{
  "total_pushed": 10000,
  "total_pulled": 9500,
  "total_acked": 9000,
  "expired_messages": 500,
  "current_pending": 1000
}
```

---

#### 5. 获取Pub/Sub统计

**端点**: `GET /api/v1/stats/pubsub`  
**认证**: 可选

**响应** (200):
```json
{
  "instance_id": "sync-svc-1",
  "publisher": {
    "total_published": 5000,
    "user_published": 4800,
    "broadcast_published": 200,
    "failed_published": 0
  },
  "subscriber": {
    "total_received": 3000,
    "user_received": 2900,
    "broadcast_received": 100,
    "processed_messages": 2950,
    "failed_messages": 50,
    "dropped_messages": 100,
    "reconnect_count": 0
  }
}
```

---

## 错误码

| 状态码 | 错误码 | 说明 |
|--------|--------|------|
| 400 | `invalid_request` | 请求格式错误 |
| 400 | `invalid_message_type` | 不支持的消息类型 |
| 401 | `unauthorized` | 未认证或Token无效 |
| 413 | `payload_too_large` | 请求体过大（>1MB） |
| 429 | `rate_limit_exceeded` | 请求频率超限 |
| 503 | `connection_limit_exceeded` | 连接数已满 |

---

## 限流规则

| API | 限制 |
|-----|------|
| WebSocket连接 | 10次/分钟 per user |
| HTTP事件API | 100次/分钟 per client |
| 离线消息API | 100次/分钟 per client |
| 统计API | 100次/分钟 per client |

---

## 客户端集成示例

### Flutter WebSocket客户端

```dart
import 'package:web_socket_channel/web_socket_channel.dart';
import 'dart:convert';
import 'dart:async';

class SyncService {
  WebSocketChannel? _channel;
  Timer? _reconnectTimer;
  int _reconnectAttempts = 0;
  final int _maxDelay = 30;

  void connect(String token) {
    final uri = Uri.parse('ws://localhost:8004/ws');
    _channel = WebSocketChannel.connect(
      uri,
      headers: {'Authorization': 'Bearer $token'},
    );

    _channel!.stream.listen(
      _onMessage,
      onError: _onError,
      onDone: _onDone,
    );

    _reconnectAttempts = 0;
  }

  void _onMessage(dynamic message) {
    final data = jsonDecode(message);
    final type = data['type'];

    switch (type) {
      case 'ping':
        // 响应Pong
        sendPong();
        break;
      case 'favorite.added':
        // 处理收藏添加
        _handleFavoriteAdded(data);
        break;
      case 'playlist.created':
        // 处理歌单创建
        _handlePlaylistCreated(data);
        break;
      // ... 其他事件类型
    }
  }

  void _onError(error) {
    print('WebSocket error: $error');
    _scheduleReconnect();
  }

  void _onDone() {
    print('WebSocket closed');
    _scheduleReconnect();
  }

  void _scheduleReconnect() {
    _reconnectTimer?.cancel();
    
    // 指数退避
    final delay = (1 << _reconnectAttempts).clamp(1, _maxDelay);
    _reconnectAttempts++;

    _reconnectTimer = Timer(Duration(seconds: delay), () {
      print('Reconnecting... (attempt $_reconnectAttempts)');
      connect(_token); // 需要保存token
    });
  }

  void sendPong() {
    final message = jsonEncode({
      'type': 'pong',
      'timestamp': DateTime.now().toIso8601String(),
    });
    _channel?.sink.add(message);
  }

  void _handleFavoriteAdded(Map<String, dynamic> data) {
    // 更新本地状态
    print('Favorite added: ${data['data']['item_id']}');
  }

  void _handlePlaylistCreated(Map<String, dynamic> data) {
    // 更新本地状态
    print('Playlist created: ${data['data']['playlist_id']}');
  }

  void dispose() {
    _reconnectTimer?.cancel();
    _channel?.sink.close();
  }
}
```

### HTTP事件推送（服务端）

```bash
# 发布单用户事件
curl -X POST http://localhost:8004/api/v1/events \
  -H "Authorization: Bearer your-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-123",
    "type": "favorite.added",
    "data": {
      "item_id": "song-456"
    }
  }'

# 批量发布
curl -X POST http://localhost:8004/api/v1/events/batch \
  -H "Authorization: Bearer your-jwt-token" \
  -H "Content-Type: application/json" \
  -d '{
    "user_ids": ["user-1", "user-2", "user-3"],
    "type": "playlist.created",
    "data": {
      "playlist_id": "pl-789"
    }
  }'
```

---

## 最佳实践

### 1. WebSocket连接管理
- 实现指数退避重连
- 处理所有消息类型（包括错误）
- 及时响应Ping/Pong
- 连接成功后拉取离线消息

### 2. 离线消息处理
- 定期拉取离线消息（上线时、定时器）
- 收到消息后立即确认（ACK）
- 使用批量ACK提高效率
- 设置超时机制（7天过期）

### 3. 错误处理
- 捕获所有WebSocket错误
- 记录错误日志
- 向用户显示友好提示
- 实现降级方案（轮询）

### 4. 性能优化
- 限制单次拉取消息数量
- 使用批量操作
- 合理设置重连间隔
- 避免频繁连接/断开

---

## 监控指标

### 关键指标

1. **连接指标**
   - `current_connections`: 当前连接数
   - `available_connections`: 可用连接数
   - `connection_limit_exceeded`: 连接被拒绝次数

2. **心跳指标**
   - `healthy_connections`: 健康连接数
   - `unhealthy_connections`: 不健康连接数
   - `avg_pong_delay`: 平均Pong延迟

3. **消息指标**
   - `total_published`: 总发布消息数
   - `failed_published`: 发布失败数
   - `dropped_messages`: 丢弃消息数（来自自己实例）

4. **离线消息指标**
   - `current_pending`: 当前待处理消息数
   - `expired_messages`: 过期消息数
   - `total_acked`: 已确认消息数

---

## 故障排查

### 常见问题

**Q: WebSocket连接立即断开**  
A: 检查JWT Token是否有效，是否达到连接限制

**Q: 收不到实时消息**  
A: 检查用户是否在线，检查Pub/Sub订阅是否正常

**Q: 离线消息不见了**  
A: 检查是否已过期（7天），检查是否已被确认

**Q: 429错误（限流）**  
A: 降低请求频率，实现客户端限流

---

## 版本历史

- **v1.0.0** (2026-03-01): 初始版本
  - WebSocket实时同步
  - 离线消息队列
  - Redis Pub/Sub跨实例广播
  - 速率限制和请求验证
