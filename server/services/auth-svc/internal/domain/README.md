# Auth Service - Domain Layer

认证服务的领域层，包含核心业务实体和业务逻辑。

## 实体说明

### User (用户实体)
**文件**: `user.go`

用户账号实体，包含以下核心功能：
- **TokenVersion**: Token版本号机制，支持密钥轮换时全局撤销所有旧Token
- **IsActive**: 账号激活状态，支持账号停用/启用
- **Validate()**: 数据验证
- **IncrementTokenVersion()**: 递增Token版本号
- **CanLogin()**: 检查用户是否可以登录

**字段**:
- `ID`: UUID
- `Phone`: 手机号（唯一）
- `TokenVersion`: Token版本号
- `IsActive`: 是否激活
- `CreatedAt`/`UpdatedAt`: 时间戳

---

### Device (设备实体)
**文件**: `device.go`

设备管理实体，用于多设备管理和异常登录检测：
- **Fingerprint**: 设备指纹机制，基于设备特征生成唯一标识
- **IsFingerprintChanged()**: 检测设备指纹是否变化（可能表示设备伪造）
- **IsSuspiciousLogin()**: 检测可疑登录行为
- **最多5台设备限制**: `MaxDevicesPerUser = 5`

**字段**:
- `ID`: UUID
- `UserID`: 所属用户ID
- `DeviceName`: 设备名称（如 "iPhone 13 Pro"）
- `Fingerprint`: 设备指纹
- `Platform`: 平台（iOS/Android/Web/Desktop）
- `AppVersion`: 应用版本
- `LastIP`: 最后登录IP
- `LastLoginAt`: 最后登录时间
- `CreatedAt`: 创建时间

**辅助函数**:
- `GenerateFingerprint()`: 生成设备指纹
- `GetDisplayName()`: 获取设备显示名称
- `IsInactive()`: 检查设备是否超过90天未登录

---

### SMSVerification (短信验证实体)
**文件**: `sms.go`

短信验证码实体，用于手机号验证：
- **6位数字验证码**: 自动生成随机验证码
- **5分钟有效期**: `SMSCodeExpiration = 5 minutes`
- **60秒发送间隔**: `SMSCodeRateLimit = 60 seconds`
- **防重复使用**: `UsedAt` 字段标记已使用的验证码

**字段**:
- `ID`: UUID
- `Phone`: 手机号
- `Code`: 6位数字验证码
- `ExpiresAt`: 过期时间
- `UsedAt`: 使用时间（可选）
- `CreatedAt`: 创建时间

**方法**:
- `NewSMSVerification()`: 创建新验证码
- `Verify()`: 验证验证码
- `IsExpired()`: 检查是否过期
- `IsUsed()`: 检查是否已使用
- `MarkAsUsed()`: 标记为已使用
- `CanResend()`: 检查是否可以重新发送

---

### SMSRecord (短信发送记录实体)
**文件**: `sms_record.go`

短信发送记录，用于统计和审计：
- **多厂商支持**: 记录使用的短信提供商（阿里云/腾讯云/Twilio）
- **成功/失败记录**: 完整的发送结果和错误信息
- **审计追踪**: 所有短信发送都有记录

**字段**:
- `ID`: UUID
- `Phone`: 手机号
- `Provider`: 提供商（aliyun/tencent/twilio）
- `Success`: 是否成功
- `ErrorMsg`: 错误信息
- `CreatedAt`: 创建时间

**静态常量**:
- `ProviderAliyun = "aliyun"`
- `ProviderTencent = "tencent"`
- `ProviderTwilio = "twilio"`

**辅助函数**:
- `NewSuccessSMSRecord()`: 创建成功记录
- `NewFailedSMSRecord()`: 创建失败记录
- `GetProviderName()`: 获取提供商中文名称

---

## 错误定义

**文件**: `errors.go`

所有领域层错误定义，包括：
- User相关错误（6个）
- Device相关错误（8个）
- SMS相关错误（8个）
- SMSRecord相关错误（2个）

---

## 使用示例

### 创建新用户
```go
user := domain.NewUser("13800138000")
if err := user.Validate(); err != nil {
    // 处理验证错误
}
```

### 注册新设备
```go
fingerprint := domain.GenerateFingerprint(deviceName, platform, deviceID, osVersion)
device := domain.NewDevice(userID, deviceName, platform, appVersion, ip, fingerprint)
if device.IsSuspiciousLogin(newFingerprint, newIP) {
    // 发送异常登录通知
}
```

### 发送短信验证码
```go
sms, err := domain.NewSMSVerification(phone)
if err != nil {
    // 处理错误
}
// 发送 sms.Code 给用户
```

### 验证短信验证码
```go
if err := sms.Verify(userInputCode); err != nil {
    switch err {
    case domain.ErrSMSCodeExpired:
        // 验证码已过期
    case domain.ErrSMSCodeInvalid:
        // 验证码错误
    case domain.ErrSMSCodeAlreadyUsed:
        // 验证码已使用
    }
}
sms.MarkAsUsed()
```

### 记录短信发送
```go
// 成功发送
record := domain.NewSuccessSMSRecord(phone, domain.ProviderAliyun)

// 发送失败
record := domain.NewFailedSMSRecord(phone, domain.ProviderTencent, "network error")
```

---

## 开发规范

1. **不可变性**: 实体创建后，关键字段（如ID、Phone）不可修改
2. **验证优先**: 所有实体都实现 `Validate()` 方法
3. **业务逻辑隔离**: 领域层只包含核心业务逻辑，不依赖外部服务
4. **错误处理**: 使用预定义的领域错误，便于上层统一处理

---

## 下一步

- [ ] 实现仓储层（Repository）
- [ ] 实现SMS服务（Service层）
- [ ] 实现JWT服务
- [ ] 添加单元测试（目标覆盖率：≥90%）
