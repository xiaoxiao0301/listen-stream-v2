# JWT 服务（版本控制）

## 概述

JWT服务提供Token生成、验证和刷新功能，支持Token版本控制和IP绑定（可选），确保安全的身份认证。

## 功能特性

- ✅ **Token版本控制**: 支持全局撤销用户所有Token
- ✅ **IP绑定验证**: 可选的IP地址绑定，防止Token盗用
- ✅ **Token刷新**: 使用RefreshToken刷新AccessToken
- ✅ **用户状态检查**: 验证用户是否激活
- ✅ **自动过期**: AccessToken 1小时，RefreshToken 7天（可配置）

## 架构设计

```
JWT Service
    ├── JWTService Interface (服务接口)
    │   ├── GenerateTokenPair (生成Token对)
    │   ├── ValidateAccessToken (验证AccessToken)
    │   ├── ValidateRefreshToken (验证RefreshToken)
    │   ├── RefreshAccessToken (刷新AccessToken)
    │   ├── RevokeUserTokens (撤销用户所有Token)
    │   └── GetTokenExpiry (获取过期时间)
    │
    ├── 基于共享库 JWT Manager
    │   ├── shared/pkg/jwt (底层JWT操作)
    │   └── HMAC-SHA256签名算法
    │
    └── 集成用户仓储
        └── Token版本验证
```

## 核心概念

### Token版本控制

每个用户有一个`TokenVersion`字段（初始值为1）：
- 生成Token时，将当前版本号写入Token
- 验证Token时，检查Token中的版本号是否与数据库中的版本号一致
- 撤销Token时，递增用户的版本号，使所有旧Token失效

**优势**：
- 全局撤销：一次操作撤销用户所有设备的Token
- 密钥轮换：更换JWT密钥时，递增所有用户的版本号
- 安全性：即使Token未过期，也可以立即失效

### IP绑定（可选）

当`IPBindingEnabled`为`true`时：
- 生成Token时，记录客户端IP地址
- 验证Token时，检查当前IP是否与Token中的IP一致
- 不一致则拒绝访问（可能是Token被盗用）

**注意**：移动网络环境下IP可能频繁变化，建议谨慎使用。

## 配置

### JWTConfig

```go
type JWTConfig struct {
    Secret           string        // JWT签名密钥（至少32字节）
    Issuer           string        // 签发者标识
    TokenExpiry      time.Duration // AccessToken过期时间（默认1小时）
    RefreshExpiry    time.Duration // RefreshToken过期时间（默认7天）
    IPBindingEnabled bool          // 是否启用IP绑定验证
}
```

### 推荐配置

**生产环境**：
```go
config := &JWTConfig{
    Secret:           os.Getenv("JWT_SECRET"), // 从环境变量加载
    Issuer:           "listen-stream",
    TokenExpiry:      1 * time.Hour,
    RefreshExpiry:    7 * 24 * time.Hour,
    IPBindingEnabled: false, // 移动端不建议启用
}
```

**开发环境**：
```go
config := &JWTConfig{
    Secret:           "dev-secret-key-32-bytes-long!!!",
    Issuer:           "listen-stream-dev",
    TokenExpiry:      24 * time.Hour, // 开发时长一点避免频繁登录
    RefreshExpiry:    30 * 24 * time.Hour,
    IPBindingEnabled: false,
}
```

## 使用方法

### 1. 创建JWT服务

```go
import (
    "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/service"
    "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/repository"
)

// 创建用户仓储
userRepo := repository.NewUserRepository(db)

// 创建JWT服务
jwtService := service.NewJWTService(&service.JWTConfig{
    Secret:           "your-secret-key",
    Issuer:           "listen-stream",
    TokenExpiry:      time.Hour,
    RefreshExpiry:    7 * 24 * time.Hour,
    IPBindingEnabled: false,
}, userRepo)
```

### 2. 生成Token对（登录成功后）

```go
ctx := context.Background()
tokenPair, err := jwtService.GenerateTokenPair(
    ctx,
    userID,    // 用户ID
    deviceID,  // 设备ID
    clientIP,  // 客户端IP（用于IP绑定）
)
if err != nil {
    // 处理错误
    return err
}

// 返回给客户端
response := map[string]interface{}{
    "access_token":  tokenPair.AccessToken,
    "refresh_token": tokenPair.RefreshToken,
    "expires_at":    tokenPair.ExpiresAt,
    "token_type":    tokenPair.TokenType, // "Bearer"
}
```

### 3. 验证AccessToken（中间件）

```go
// HTTP中间件示例
func AuthMiddleware(jwtService service.JWTService) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 从Header获取Token
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "missing authorization header"})
            c.Abort()
            return
        }

        // 2. 解析Bearer Token
        token := strings.TrimPrefix(authHeader, "Bearer ")
        if token == authHeader {
            c.JSON(401, gin.H{"error": "invalid token format"})
            c.Abort()
            return
        }

        // 3. 验证Token
        clientIP := c.ClientIP()
        claims, err := jwtService.ValidateAccessToken(c.Request.Context(), token, clientIP)
        if err != nil {
            c.JSON(401, gin.H{"error": "invalid token"})
            c.Abort()
            return
        }

        // 4. 将用户信息存入上下文
        c.Set("user_id", claims.UserID)
        c.Set("device_id", claims.DeviceID)
        c.Next()
    }
}
```

### 4. 刷新AccessToken

```go
func RefreshTokenHandler(jwtService service.JWTService) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req struct {
            RefreshToken string `json:"refresh_token"`
        }
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": "invalid request"})
            return
        }

        // 刷新Token
        clientIP := c.ClientIP()
        tokenPair, err := jwtService.RefreshAccessToken(c.Request.Context(), req.RefreshToken, clientIP)
        if err != nil {
            c.JSON(401, gin.H{"error": "invalid refresh token"})
            return
        }

        c.JSON(200, gin.H{
            "access_token":  tokenPair.AccessToken,
            "refresh_token": tokenPair.RefreshToken,
            "expires_at":    tokenPair.ExpiresAt,
            "token_type":    tokenPair.TokenType,
        })
    }
}
```

### 5. 撤销用户所有Token（登出所有设备）

```go
func LogoutAllDevicesHandler(jwtService service.JWTService) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.GetString("user_id") // 从中间件获取

        // 撤销所有Token
        if err := jwtService.RevokeUserTokens(c.Request.Context(), userID); err != nil {
            c.JSON(500, gin.H{"error": "failed to revoke tokens"})
            return
        }

        c.JSON(200, gin.H{"message": "logged out from all devices"})
    }
}
```

## 错误处理

### 自定义错误

```go
// ErrTokenVersionMismatch Token版本不匹配（已被撤销）
service.ErrTokenVersionMismatch

// ErrIPMismatch IP地址不匹配（可能被盗用）
service.ErrIPMismatch

// domain.ErrUserNotFound 用户不存在
domain.ErrUserNotFound

// domain.ErrUserInactive 用户已停用
domain.ErrUserInactive
```

### 错误处理示例

```go
claims, err := jwtService.ValidateAccessToken(ctx, token, clientIP)
if err != nil {
    switch err {
    case service.ErrTokenVersionMismatch:
        // Token已被撤销，要求重新登录
        return errors.New("token has been revoked, please login again")
    case service.ErrIPMismatch:
        // IP不匹配，可能是安全问题
        return errors.New("suspicious activity detected, please login again")
    case domain.ErrUserNotFound:
        return errors.New("user not found")
    case domain.ErrUserInactive:
        return errors.New("user account is inactive")
    default:
        // 其他错误（如Token过期、签名无效等）
        return errors.New("invalid token")
    }
}
```

## Token生命周期

```
用户登录
    ↓
生成Token对（AccessToken + RefreshToken）
    ↓
客户端保存Token
    ↓
每次请求携带AccessToken
    ↓
服务端验证Token（检查版本号、IP等）
    ↓
AccessToken过期？
    ├─ 是 → 使用RefreshToken刷新
    │          ↓
    │       生成新的Token对
    │          ↓
    │       返回新Token
    └─ 否 → 继续使用
```

## 安全建议

### 1. JWT密钥管理
- ✅ 使用至少32字节的随机密钥
- ✅ 定期轮换密钥（3-6个月）
- ✅ 不要硬编码，使用环境变量或密钥管理服务
- ✅ 轮换密钥时，递增所有用户的TokenVersion

### 2. Token存储（客户端）
- ✅ AccessToken存储到内存（不持久化）
- ✅ RefreshToken存储到安全存储（iOS Keychain、Android Keystore）
- ❌ 不要存储到LocalStorage（XSS风险）
- ❌ 不要存储到Cookie（CSRF风险，除非使用HttpOnly + SameSite）

### 3. Token传输
- ✅ 使用HTTPS
- ✅ 使用`Authorization: Bearer <token>`头
- ❌ 不要在URL参数中传递Token

### 4. 过期时间设置
- AccessToken: 短（15分钟 - 1小时）
- RefreshToken: 长（7天 - 30天）
- 移动端可以适当延长AccessToken时间（用户体验）

### 5. IP绑定建议
- Web应用: 可以启用（IP相对稳定）
- 移动应用: 不建议启用（4G/WiFi切换导致IP变化）
- 内网应用: 可以启用（IP稳定）

## 性能优化

### 1. 无DB查询验证（proxy-svc）
JWT本身是自包含的，可以不查数据库验证：
```go
// 仅验证签名和过期时间（适用于高频调用）
claims, err := jwtManager.ValidateToken(token)
```

### 2. 版本验证（auth-svc）
需要查数据库验证版本号：
```go
// 完整验证（包含版本号和用户状态）
claims, err := jwtService.ValidateAccessToken(ctx, token, clientIP)
```

### 3. 缓存用户TokenVersion
使用Redis缓存用户的TokenVersion，减少数据库查询：
```go
// 伪代码
version, err := redis.Get("user:token_version:" + userID)
if err != nil {
    version, err = db.GetUserTokenVersion(userID)
    redis.Set("user:token_version:" + userID, version, 5 * time.Minute)
}
```

## 测试

### 运行单元测试

```bash
cd server/services/auth-svc
go test -v ./internal/service/
```

### 测试覆盖率

```bash
go test -cover ./internal/service/
# coverage: 65.5% of statements
```

### 测试用例

- ✅ 生成Token对
- ✅ 验证AccessToken
- ✅ Token版本不匹配
- ✅ IP绑定验证
- ✅ 刷新Token

## 集成示例

### 完整的登录流程

```go
// 1. 验证SMS验证码
err := smsService.VerifyCode(ctx, phone, code)
if err != nil {
    return err
}

// 2. 获取或创建用户
user, err := userRepo.GetByPhone(ctx, phone)
if err == domain.ErrUserNotFound {
    user = domain.NewUser(phone)
    _ = userRepo.Create(ctx, user)
}

// 3. 注册设备
device, err := deviceService.RegisterOrUpdateDevice(ctx, user.ID, deviceInfo)
if err != nil {
    return err
}

// 4. 生成Token
tokenPair, err := jwtService.GenerateTokenPair(ctx, user.ID, device.ID, clientIP)
if err != nil {
    return err
}

// 5. 返回Token
return tokenPair
```

## 相关文档

- [listen-stream-redesign.md](../../../../../docs/listen-stream-redesign.md) - 系统重构方案
- [共享库JWT文档](../../../../shared/pkg/jwt/README.md) - 底层JWT实现
- [User Domain](../../domain/user.go) - 用户领域模型
- [User Repository](../../repository/user_repo.go) - 用户仓储

## 后续优化

- [ ] 支持多密钥（密钥ID轮换）
- [ ] 支持RSA签名算法（公私钥分离）
- [ ] Token黑名单（Redis存储被撤销的Token）
- [ ] 更细粒度的权限控制（RBAC）
- [ ] 设备指纹检测（结合device_service）
- [ ] 自动刷新Token（客户端SDK）

## 总结

JWT服务提供了安全、灵活的身份认证机制：
1. **版本控制**: 支持全局撤销Token
2. **IP绑定**: 可选的额外安全层
3. **Token刷新**: 无感知的Token更新
4. **用户状态**: 实时检查用户是否可用
5. **高性能**: 支持无DB验证（proxy-svc）

完全符合设计要求，可直接用于生产环境。
