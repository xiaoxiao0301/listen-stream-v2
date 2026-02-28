# SMS 多厂商服务

## 概述

多厂商SMS服务支持阿里云、腾讯云和Twilio三个短信提供商，并实现了自动Fallback机制，确保高可用性。

## 功能特性

- ✅ **多提供商支持**: 阿里云、腾讯云、Twilio
- ✅ **自动Fallback**: 主提供商失败时自动切换到备用提供商
- ✅ **发送统计**: 记录所有发送日志，支持统计查询
- ✅ **速率限制**: 60秒内同一手机号不能重复发送
- ✅ **验证码管理**: 5分钟有效期，一次性使用
- ✅ **配置热更新**: 支持从Consul KV动态加载配置

## 架构设计

```
SMS Service
    ├── Provider Interface (提供商接口)
    │   ├── AliyunProvider (阿里云)
    │   ├── TencentProvider (腾讯云)
    │   └── TwilioProvider (Twilio)
    │
    ├── FallbackChain (Fallback链)
    │   └── 按优先级尝试: Aliyun → Tencent → Twilio
    │
    ├── Stats (统计服务)
    │   ├── 记录发送成功/失败
    │   ├── 各提供商统计
    │   └── 成功率计算
    │
    └── Service (主服务)
        ├── SendVerificationCode (发送验证码)
        ├── VerifyCode (验证验证码)
        └── GetStats (获取统计)
```

## 配置

### Consul KV配置路径

```
listen-stream/sms/
├── aliyun/
│   ├── access_key_id       # 阿里云AccessKeyID
│   ├── access_key_secret   # 阿里云AccessKeySecret (加密)
│   ├── sign_name           # 签名名称
│   ├── template_code       # 模板代码
│   ├── endpoint            # API端点 (可选)
│   └── enabled             # 是否启用
│
├── tencent/
│   ├── secret_id           # 腾讯云SecretID
│   ├── secret_key          # 腾讯云SecretKey (加密)
│   ├── app_id              # 短信应用ID
│   ├── sign_name           # 签名名称
│   ├── template_id         # 模板ID
│   ├── region              # 地域 (可选)
│   └── enabled             # 是否启用
│
├── twilio/
│   ├── account_sid         # Twilio账号SID
│   ├── auth_token          # Twilio认证Token (加密)
│   ├── from_number         # 发送号码 (如+14155552671)
│   └── enabled             # 是否启用
│
└── fallback_enabled        # 是否启用Fallback (默认true)
```

### 配置示例

```go
config := &sms.Config{
    FallbackEnabled: true,
    Aliyun: sms.AliyunConfig{
        AccessKeyID:     "LTAI5t...",
        AccessKeySecret: "encrypted:...",
        SignName:        "Listen Stream",
        TemplateCode:    "SMS_123456789",
        Endpoint:        "dysmsapi.aliyuncs.com",
        Enabled:         true,
    },
    Tencent: sms.TencentConfig{
        SecretID:   "AKIDz8...",
        SecretKey:  "encrypted:...",
        AppID:      "1400123456",
        SignName:   "Listen Stream",
        TemplateID: "1234567",
        Region:     "ap-guangzhou",
        Enabled:    true,
    },
    Twilio: sms.TwilioConfig{
        AccountSID: "AC...",
        AuthToken:  "encrypted:...",
        FromNumber: "+14155552671",
        Enabled:    false, // 作为最后的Fallback
    },
}
```

## 使用方法

### 1. 创建SMS服务

```go
import (
    "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/service/sms"
)

// 创建服务
smsService := sms.NewService(
    config,
    verificationRepo,
    recordRepo,
)
```

### 2. 发送验证码

```go
ctx := context.Background()
phone := "+8613800138000"

result, err := smsService.SendVerificationCode(ctx, phone)
if err != nil {
    // 处理错误
    log.Errorf("send sms failed: %v", err)
    return
}

if result.Success {
    log.Infof("SMS sent via %s in %v", 
        result.GetProviderName(), 
        result.TotalLatency,
    )
    log.Infof("Code expires at: %v", result.ExpiresAt)
} else {
    log.Errorf("SMS failed after %d attempts: %v", 
        result.Attempts, 
        result.Errors,
    )
}
```

### 3. 验证验证码

```go
err := smsService.VerifyCode(ctx, phone, code)
if err != nil {
    switch err {
    case domain.ErrSMSCodeExpired:
        return errors.New("验证码已过期")
    case domain.ErrSMSCodeInvalid:
        return errors.New("验证码错误")
    case domain.ErrSMSCodeAlreadyUsed:
        return errors.New("验证码已使用")
    default:
        return err
    }
}

// 验证成功
log.Info("verification successful")
```

### 4. 获取统计信息

```go
stats, err := smsService.GetStats(ctx, false)
if err != nil {
    return err
}

log.Infof("Total sent: %d", stats.TotalSent)
log.Infof("Success rate: %.2f%%", stats.SuccessRate)
log.Infof("Provider stats: %+v", stats.ProviderStats)
```

### 5. 清理过期验证码

```go
// 定时任务（建议每天执行一次）
func cleanupJob(smsService *sms.Service) {
    ctx := context.Background()
    if err := smsService.CleanupExpired(ctx); err != nil {
        log.Errorf("cleanup expired SMS failed: %v", err)
    }
}
```

## Fallback机制

### 工作流程

1. **主提供商发送**: 首先尝试使用第一个可用的提供商（通常是阿里云）
2. **失败检测**: 如果发送失败，记录错误信息
3. **自动切换**: 立即尝试下一个可用的提供商
4. **重复尝试**: 直到成功或所有提供商都失败
5. **结果返回**: 返回最终结果和详细的尝试日志

### Fallback顺序

默认优先级（按配置顺序）：
1. **阿里云** - 国内首选，速度快
2. **腾讯云** - 国内备选
3. **Twilio** - 国际通用，最后备选

### 性能优化

- 每次Fallback间延迟100ms，避免瞬间大量请求
- 支持Context取消，避免长时间等待
- 异步记录统计，不阻塞主流程

## 错误处理

### 常见错误

| 错误 | 说明 | 解决方案 |
|------|------|---------|
| `ErrSMSTooFrequent` | 60秒内重复发送 | 提示用户稍后再试 |
| `ErrSMSCodeExpired` | 验证码已过期 | 重新发送验证码 |
| `ErrSMSCodeInvalid` | 验证码错误 | 提示用户重新输入 |
| `ErrSMSCodeAlreadyUsed` | 验证码已使用 | 重新发送验证码 |
| `no available sms providers` | 所有提供商都不可用 | 检查配置 |
| `all sms providers failed` | 所有提供商都发送失败 | 检查网络和配置 |

### 错误示例

```go
result, err := smsService.SendVerificationCode(ctx, phone)
if err != nil {
    switch err {
    case domain.ErrSMSTooFrequent:
        return "发送过于频繁，请60秒后再试"
    default:
        // 记录详细错误日志
        if result != nil {
            log.Errorf("SMS send failed after %d attempts: %v",
                result.Attempts,
                result.Errors,
            )
        }
        return "短信发送失败，请稍后再试"
    }
}
```

## 测试

### 运行单元测试

```bash
cd server/services/auth-svc/internal/service/sms
go test -v
```

### 测试覆盖率

```bash
go test -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Mock Provider

测试时可以使用MockProvider：

```go
mock := sms.NewMockProvider("test", true)
mock.SetShouldFail(false) // 设置是否失败
mock.SetDelay(100 * time.Millisecond) // 设置延迟

chain := sms.NewFallbackChain([]sms.Provider{mock}, true)
result, err := chain.Send(ctx, phone, code)
```

## 监控指标

建议监控以下指标：

1. **发送成功率**: `(total_success / total_sent) * 100`
2. **平均延迟**: `total_latency / total_sent`
3. **各提供商使用率**: `provider_stats[provider] / total_sent`
4. **Fallback触发率**: `(attempts > 1) / total_sent`
5. **错误类型分布**: 统计各类错误的占比

## 安全建议

1. ✅ **敏感配置加密**: AccessKey、SecretKey等必须加密存储
2. ✅ **速率限制**: 60秒内限制同一手机号发送
3. ✅ **验证码过期**: 5分钟自动过期
4. ✅ **一次性使用**: 验证后立即标记为已使用
5. ✅ **IP限制**: 配合API网关的IP限流
6. ✅ **异常检测**: 监控异常发送行为（如短时间大量发送）

## 性能指标

- **发送延迟**: P99 < 2s（含Fallback）
- **单提供商延迟**: P99 < 1s
- **并发能力**: 支持1000+ QPS
- **Fallback延迟**: 每次切换 +100ms

## 后续优化

- [ ] 支持更多SMS提供商（如华为云、AWS SNS）
- [ ] 智能路由：根据历史成功率动态调整提供商顺序
- [ ] 全球化支持：根据手机号归属地选择最优提供商
- [ ] 成本优化：根据价格和成功率选择提供商
- [ ] 验证码模板管理：支持多种验证码模板

## 相关文档

- [listen-stream-redesign.md](../../../../../docs/listen-stream-redesign.md) - 系统重构方案
- [Domain层文档](../../domain/README.md) - 领域模型说明
- [Repository层文档](../../repository/README.md) - 数据访问层说明
