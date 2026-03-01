-- 001_create_admin_tables.up.sql

-- 管理员用户表
CREATE TABLE IF NOT EXISTS admin_users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'viewer',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    totp_secret VARCHAR(255),
    totp_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    last_login_at TIMESTAMP,
    last_login_ip VARCHAR(45),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_admin_users_username ON admin_users(username);
CREATE INDEX idx_admin_users_email ON admin_users(email);
CREATE INDEX idx_admin_users_status ON admin_users(status);

-- 操作日志表
CREATE TABLE IF NOT EXISTS operation_logs (
    id VARCHAR(36) PRIMARY KEY,
    admin_id VARCHAR(36) NOT NULL,
    admin_name VARCHAR(50) NOT NULL,
    operation VARCHAR(50) NOT NULL,
    resource VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255),
    action VARCHAR(20) NOT NULL,
    details JSONB,
    request_id VARCHAR(36),
    ip VARCHAR(45) NOT NULL,
    user_agent TEXT,
    status VARCHAR(20) NOT NULL,
    error_msg TEXT,
    duration BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_operation_logs_admin_id ON operation_logs(admin_id);
CREATE INDEX idx_operation_logs_operation ON operation_logs(operation);
CREATE INDEX idx_operation_logs_resource ON operation_logs(resource);
CREATE INDEX idx_operation_logs_status ON operation_logs(status);
CREATE INDEX idx_operation_logs_created_at ON operation_logs(created_at DESC);
CREATE INDEX idx_operation_logs_request_id ON operation_logs(request_id);

-- 每日统计表
CREATE TABLE IF NOT EXISTS daily_stats (
    date DATE PRIMARY KEY,
    total_users BIGINT NOT NULL DEFAULT 0,
    new_users BIGINT NOT NULL DEFAULT 0,
    active_users BIGINT NOT NULL DEFAULT 0,
    total_requests BIGINT NOT NULL DEFAULT 0,
    success_requests BIGINT NOT NULL DEFAULT 0,
    failed_requests BIGINT NOT NULL DEFAULT 0,
    error_rate DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    avg_response_time BIGINT NOT NULL DEFAULT 0,
    total_favorites BIGINT NOT NULL DEFAULT 0,
    total_playlists BIGINT NOT NULL DEFAULT 0,
    total_plays BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_daily_stats_date ON daily_stats(date DESC);

-- 异常活动表
CREATE TABLE IF NOT EXISTS anomalous_activities (
    id VARCHAR(36) PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    description TEXT NOT NULL,
    admin_id VARCHAR(36) NOT NULL,
    admin_name VARCHAR(50) NOT NULL,
    details TEXT,
    resolved BOOLEAN NOT NULL DEFAULT FALSE,
    resolved_by VARCHAR(36),
    resolved_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_anomalous_activities_admin_id ON anomalous_activities(admin_id);
CREATE INDEX idx_anomalous_activities_type ON anomalous_activities(type);
CREATE INDEX idx_anomalous_activities_severity ON anomalous_activities(severity);
CREATE INDEX idx_anomalous_activities_resolved ON anomalous_activities(resolved);
CREATE INDEX idx_anomalous_activities_created_at ON anomalous_activities(created_at DESC);

-- 配置变更历史表
CREATE TABLE IF NOT EXISTS config_histories (
    id VARCHAR(36) PRIMARY KEY,
    config_key VARCHAR(255) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    version BIGINT NOT NULL,
    admin_id VARCHAR(36) NOT NULL,
    admin_name VARCHAR(50) NOT NULL,
    reason TEXT,
    rollbackable BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_config_histories_config_key ON config_histories(config_key);
CREATE INDEX idx_config_histories_admin_id ON config_histories(admin_id);
CREATE INDEX idx_config_histories_created_at ON config_histories(created_at DESC);
CREATE INDEX idx_config_histories_version ON config_histories(version DESC);
