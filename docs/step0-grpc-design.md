# 步骤0: Protobuf接口定义 + gRPC封装层

## 🎯 目标
建立服务间通信的基础设施，为所有微服务提供高性能、类型安全的RPC通信能力。

## 📐 架构决策

### 为什么选择gRPC？
1. **性能**: Protobuf序列化比JSON快10x，HTTP/2多路复用
2. **类型安全**: .proto自动生成代码，编译时发现错误
3. **流式传输**: 支持双向流，适合实时场景
4. **代码生成**: 自动生成客户端/服务端代码，减少手写

### 混合架构设计
```
客户端(Flutter/React) 
       ↓ HTTP REST + JSON (易用性优先)
    proxy-svc (API网关)
       ↓ gRPC + Protobuf (性能优先)
auth-svc / user-svc / sync-svc
```

## 📁 目录结构

```
server/
├── shared/
│   ├── proto/                     # Protobuf定义
│   │   ├── auth/v1/
│   │   │   └── auth.proto        # 认证服务接口
│   │   ├── user/v1/
│   │   │   └── user.proto        # 用户内容服务接口
│   │   ├── sync/v1/
│   │   │   └── sync.proto        # 同步事件服务接口
│   │   ├── admin/v1/
│   │   │   └── admin.proto       # 管理服务接口
│   │   ├── buf.yaml              # Buf配置(推荐)
│   │   └── Makefile              # protoc生成命令
│   │
│   ├── gen/                       # 生成的Go代码
│   │   └── proto/
│   │       ├── auth/v1/          # authv1包
│   │       ├── user/v1/          # userv1包
│   │       ├── sync/v1/          # syncv1包
│   │       └──admin/v1/         # adminv1包
│   │
│   └── pkg/
│       └── grpc/                  # gRPC封装toolkit
│           ├── client.go          # 客户端工具
│           ├── server.go          # 服务端工具
│           ├── interceptor/
│           │   ├── logging.go     # 日志拦截器
│           │   ├── tracing.go     # 追踪拦截器
│           │   ├── recovery.go    # Panic恢复
│           │   ├── auth.go        # JWT验证
│           │   └── ratelimit.go   # 限流
│           ├── errors.go          # 错误处理
│           ├── metadata.go        # 元数据工具
│           └── health.go          # 健康检查
```

## 📝 Protobuf接口定义

### auth.proto (认证服务)

```protobuf
syntax = "proto3";
package auth.v1;
option go_package = "github.com/yourorg/listen-stream/gen/proto/auth/v1;authv1";

import "google/protobuf/timestamp.proto";

// AuthService 认证服务
// 提供Token验证、刷新、撤销等功能
service AuthService {
  // VerifyToken 验证访问Token
  // 高频调用：proxy-svc每个请求都会调用
  // 性能要求：P99 < 5ms
  rpc VerifyToken(VerifyTokenRequest) returns (VerifyTokenResponse);
  
  // RefreshToken 刷新Token
  rpc RefreshToken(RefreshTokenRequest) returns (RefreshTokenResponse);
  
  // RevokeToken 撤销Token(用户登出)
  rpc RevokeToken(RevokeTokenRequest) returns (RevokeTokenResponse);
  
  // RevokeDevice 踢出设备
  rpc RevokeDevice(RevokeDeviceRequest) returns (RevokeDeviceResponse);
}

message VerifyTokenRequest {
  string access_token = 1;          // 访问Token
  string client_ip = 2;             // 客户端IP(可选，严格模式)
  string device_fingerprint = 3;    // 设备指纹(可选)
}

message VerifyTokenResponse {
  bool valid = 1;                   // 是否有效
  string user_id = 2;               // 用户ID
  string role = 3;                  // 角色: USER/ADMIN/SUPER_ADMIN
  int32 token_version = 4;          // Token版本号
  google.protobuf.Timestamp expires_at = 5;  // 过期时间
  ErrorCode error_code = 6;         // 错误码
  string error_message = 7;         // 错误消息
}

enum ErrorCode {
  ERROR_CODE_UNSPECIFIED = 0;       // 未指定
  ERROR_CODE_TOKEN_EXPIRED = 1;     // Token过期
  ERROR_CODE_TOKEN_INVALID = 2;     // Token无效
  ERROR_CODE_TOKEN_REVOKED = 3;     // Token已撤销
  ERROR_CODE_VERSION_MISMATCH = 4;  // 版本不匹配
  ERROR_CODE_IP_MISMATCH = 5;       // IP不匹配
}

message RefreshTokenRequest {
  string refresh_token = 1;
  string device_id = 2;
}

message RefreshTokenResponse {
  string access_token = 1;
  string refresh_token = 2;
  google.protobuf.Timestamp access_token_expires_at = 3;
  google.protobuf.Timestamp refresh_token_expires_at = 4;
}

message RevokeTokenRequest {
  string user_id = 1;
  string device_id = 2;
}

message RevokeTokenResponse {
  bool success = 1;
}

message RevokeDeviceRequest {
  string user_id = 1;
  string device_id = 2;
}

message RevokeDeviceResponse {
  bool success = 1;
  int32 remaining_devices = 2;       // 剩余设备数
}
```

### user.proto (用户内容服务)

```protobuf
syntax = "proto3";
package user.v1;
option go_package = "github.com/yourorg/listen-stream/gen/proto/user/v1;userv1";

import "google/protobuf/timestamp.proto";

// UserService 用户内容服务
// 管理收藏、歌单、播放历史
service UserService {
  // 收藏管理
  rpc AddFavorite(AddFavoriteRequest) returns (AddFavoriteResponse);
  rpc RemoveFavorite(RemoveFavoriteRequest) returns (RemoveFavoriteResponse);
  rpc ListFavorites(ListFavoritesRequest) returns (ListFavoritesResponse);
  
  // 播放历史
  rpc AddHistory(AddHistoryRequest) returns (AddHistoryResponse);
  rpc ListHistory(ListHistoryRequest) returns (ListHistoryResponse);
  
  // 歌单管理
  rpc CreatePlaylist(CreatePlaylistRequest) returns (CreatePlaylistResponse);
  rpc UpdatePlaylist(UpdatePlaylistRequest) returns (UpdatePlaylistResponse);
  rpc DeletePlaylist(DeletePlaylistRequest) returns (DeletePlaylistResponse);
  rpc ListPlaylists(ListPlaylistsRequest) returns (ListPlaylistsResponse);
}

message Favorite {
  string id = 1;
  string user_id = 2;
  FavoriteType type = 3;
  string target_id = 4;
  google.protobuf.Timestamp created_at = 5;
}

enum FavoriteType {
  FAVORITE_TYPE_UNSPECIFIED = 0;
  FAVORITE_TYPE_SONG = 1;
  FAVORITE_TYPE_ALBUM = 2;
  FAVORITE_TYPE_ARTIST = 3;
  FAVORITE_TYPE_MV = 4;
}

message AddFavoriteRequest {
  string user_id = 1;
  FavoriteType type = 2;
  string target_id = 3;
}

message AddFavoriteResponse {
  Favorite favorite = 1;
}

// ... 其他消息定义省略，实际实现时需补充完整
```

## 🔧 gRPC工具库实现

### client.go

```go
package grpc

import (
    "context"
    "time"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

type ClientConfig struct {
    ServiceName string        // 服务名(用于服务发现)
    Address    string        // 直连地址(可选)
    Timeout     time.Duration // 超时时间
    MaxRetries  int          // 重试次数
}

// NewClient 创建gRPC客户端连接
func NewClient(ctx context.Context, cfg ClientConfig) (*grpc.ClientConn, error) {
    opts := []grpc.DialOption{
        // 生产环境应使用TLS
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        
        // OpenTelemetry追踪
        grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
        
        // 负载均衡
        grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
        
        // 拦截器链
        grpc.WithChainUnaryInterceptor(
            LoggingUnaryClientInterceptor(),
            RetryUnaryClientInterceptor(cfg.MaxRetries),
        ),
        
        // KeepAlive设置
        grpc.WithKeepaliveParams(keepalive.ClientParameters{
            Time:                30 * time.Second,
            Timeout:             10 * time.Second,
            PermitWithoutStream: true,
        }),
    }
    
    // 服务发现: 使用Consul DNS
    target := cfg.Address
    if target == "" {
        target = fmt.Sprintf("%s.service.consul:%d", cfg.ServiceName, 9001)
    }
    
    conn, err := grpc.DialContext(ctx, target, opts...)
    if err != nil {
        return nil, err
    }
    
    return conn, nil
}
```

### server.go

```go
package grpc

import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"
    "google.golang.org/grpc/reflection"
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

type ServerConfig struct {
    Port            int
    MaxConnections  int
    EnableReflection bool // 开发环境启用，便于grpcurl调试
}

// NewServer 创建gRPC服务器
func NewServer(cfg ServerConfig) *grpc.Server {
    opts := []grpc.ServerOption{
        // OpenTelemetry追踪
        grpc.StatsHandler(otelgrpc.NewServerHandler()),
        
        // 拦截器链(注意顺序)
        grpc.ChainUnaryInterceptor(
            RecoveryUnaryServerInterceptor(),    // 最外层：捕获panic
            LoggingUnaryServerInterceptor(),     // 日志
            TracingUnaryServerInterceptor(),     // 追踪
            // AuthUnaryServerInterceptor(),     // 认证(可选)
        ),
        
        grpc.ChainStreamInterceptor(
            RecoveryStreamServerInterceptor(),
            LoggingStreamServerInterceptor(),
        ),
        
        // 最大消息大小
        grpc.MaxRecvMsgSize(10 * 1024 * 1024),  // 10MB
        grpc.MaxSendMsgSize(10 * 1024 * 1024),
        
        // 连接数限制
        grpc.MaxConcurrentStreams(uint32(cfg.MaxConnections)),
    }
    
    srv := grpc.NewServer(opts...)
    
    // 注册健康检查
    healthSrv := health.NewServer()
    grpc_health_v1.RegisterHealthServer(srv, healthSrv)
    
    // 开发环境启用反射(grpcurl需要)
    if cfg.EnableReflection {
        reflection.Register(srv)
    }
    
    return srv
}
```

### 拦截器实现(示例)

```go
// interceptor/logging.go
package interceptor

import (
    "context"
    "time"
    
    "go.uber.org/zap"
    "google.golang.org/grpc"
)

func LoggingUnaryServerInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        start := time.Now()
        
        // 调用处理器
        resp, err := handler(ctx, req)
        
        // 记录日志
        duration := time.Since(start)
        logger.Info("grpc request",
            zap.String("method", info.FullMethod),
            zap.Duration("duration", duration),
            zap.Error(err),
        )
        
        return resp, err
    }
}
```

## ✅ 完成标准

### Protobuf定义
- [ ] auth.proto 完整定义(4个RPC方法)
- [ ] user.proto 完整定义(9个RPC方法)
- [ ] sync.proto 完整定义(3个RPC方法)
- [ ] admin.proto 完整定义(管理接口)
- [ ] 代码生成成功(`make proto-gen`)
- [ ] 生成的Go代码无编译错误

### gRPC工具库
- [ ] 客户端工具实现(连接池、超时、重试)
- [ ] 服务器工具实现(配置、优雅关闭)
- [ ] 日志拦截器(结构化日志 + RequestID)
- [ ] 追踪拦截器(OpenTelemetry集成)
- [ ] 恢复拦截器(Panic捕获 + 日志)
- [ ] 错误转换工具(Status ↔ 业务错误)

### 测试
- [ ] 单元测试覆盖率 ≥ 80%
- [ ] Mock服务器/客户端测试
- [ ] 拦截器正确执行
- [ ] 错误传播正确
- [ ] Panic被捕获

### 性能验证
- [ ] gRPC调用延迟 < 5ms(同机房)
- [ ] 单连接QPS > 10000(简单RPC)
- [ ] 压测: 1000并发稳定运行
- [ ] 内存无泄漏

### 文档
- [ ] .proto文件注释完整
- [ ] gRPC工具库使用文档
- [ ] 示例代码(客户端+服务端)

## 🚀 预期收益

| 指标 | HTTP REST | gRPC | 提升 |
|------|-----------|------|------|
| P99延迟 | 50ms | 5ms | **10x** |
| 吞吐量(QPS) | 2000 | 15000 | **7.5x** |
| 序列化开销 | ~1ms | ~0.1ms | **10x** |
| 代码安全 | 手动定义 | 类型安全 | ✅ |

**关键路径优化**:
- `proxy-svc → auth-svc.VerifyToken`: 10000+次/秒，延迟从50ms→5ms
- `proxy-svc → user-svc.ListFavorites`: 批量查询性能提升5x

## ⚠️ 风险与对策

| 风险 | 影响 | 对策 |
|------|------|------|
| 接口设计不当 | 后期难改 | 充分评审，迭代设计 |
| 版本兼容性 | 升级困难 | 遵循向后兼容原则 |
| 调试困难 | 开发效率 | 启用反射，使用grpcurl/BloomRPC |
| 连接泄漏 | 资源耗尽 | defer关闭，监控连接数 |
| 消息过大 | 传输失败 | 调整MaxMessageSize，使用流式RPC |

---

**完成时间估计**: 1-2天
**前置依赖**: 无
**后续步骤**: 步骤1(crypto工具库)
