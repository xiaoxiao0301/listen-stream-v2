-- 001_create_auth_tables.up.sql
-- 认证服务数据库表

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    phone VARCHAR(11) NOT NULL UNIQUE,
    token_version INTEGER NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 为手机号创建索引
CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_is_active ON users(is_active);

-- 设备表
CREATE TABLE IF NOT EXISTS devices (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    device_name VARCHAR(100) NOT NULL,
    fingerprint VARCHAR(64) NOT NULL,
    platform VARCHAR(20) NOT NULL,
    app_version VARCHAR(20) NOT NULL,
    last_ip VARCHAR(45) NOT NULL,
    last_login_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- 为用户ID创建索引
CREATE INDEX idx_devices_user_id ON devices(user_id);
CREATE INDEX idx_devices_fingerprint ON devices(fingerprint);
CREATE INDEX idx_devices_last_login_at ON devices(last_login_at);

-- 短信验证表
CREATE TABLE IF NOT EXISTS sms_verifications (
    id VARCHAR(36) PRIMARY KEY,
    phone VARCHAR(11) NOT NULL,
    code VARCHAR(6) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 为手机号创建索引
CREATE INDEX idx_sms_verifications_phone ON sms_verifications(phone);
CREATE INDEX idx_sms_verifications_expires_at ON sms_verifications(expires_at);
CREATE INDEX idx_sms_verifications_created_at ON sms_verifications(created_at);

-- 短信发送记录表
CREATE TABLE IF NOT EXISTS sms_records (
    id VARCHAR(36) PRIMARY KEY,
    phone VARCHAR(11) NOT NULL,
    provider VARCHAR(20) NOT NULL,
    success BOOLEAN NOT NULL,
    error_msg TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 为手机号和创建时间创建索引
CREATE INDEX idx_sms_records_phone ON sms_records(phone);
CREATE INDEX idx_sms_records_created_at ON sms_records(created_at);
CREATE INDEX idx_sms_records_provider ON sms_records(provider);

-- 添加注释
COMMENT ON TABLE users IS '用户表';
COMMENT ON TABLE devices IS '设备表';
COMMENT ON TABLE sms_verifications IS '短信验证表';
COMMENT ON TABLE sms_records IS '短信发送记录表';
