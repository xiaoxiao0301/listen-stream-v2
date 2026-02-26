# Consul KV 配置初始化脚本
# 用途: 首次部署时初始化Consul KV中的业务配置
# 使用: ./scripts/init-consul-config.sh

#!/bin/bash

CONSUL_ADDR="${CONSUL_ADDR:-localhost:8500}"
NAMESPACE="listen-stream"

echo "Initializing Consul KV configuration..."
echo "Consul address: $CONSUL_ADDR"
echo "Namespace: $NAMESPACE"
echo ""

# ===== 通用配置 =====
echo "Setting common configuration..."

# JWT配置
consul kv put ${NAMESPACE}/common/jwt_secret "$(openssl rand -base64 32)"
consul kv put ${NAMESPACE}/common/jwt_version "1"
consul kv put ${NAMESPACE}/common/jwt_access_token_ttl "3600"      # 1小时
consul kv put ${NAMESPACE}/common/jwt_refresh_token_ttl "2592000"  # 30天

# AES加密密钥(用于加密Consul中的敏感配置)
consul kv put ${NAMESPACE}/common/aes_key "$(openssl rand -base64 32)"

echo "✓ Common configuration set"
echo ""

# ===== 第三方API配置 =====
echo "Setting API configuration..."

# QQ音乐API
consul kv put ${NAMESPACE}/api/qq_music/base_url "https://u.y.qq.com"
consul kv put ${NAMESPACE}/api/qq_music/api_key "your_qq_music_api_key"
consul kv put ${NAMESPACE}/api/qq_music/cookie "your_cookie_here"
consul kv put ${NAMESPACE}/api/qq_music/enabled "true"
consul kv put ${NAMESPACE}/api/qq_music/timeout "10s"
consul kv put ${NAMESPACE}/api/qq_music/rate_limit "20"  # 20 req/s

# Joox API (Fallback 1)
consul kv put ${NAMESPACE}/api/joox/base_url "https://api.joox.com"
consul kv put ${NAMESPACE}/api/joox/enabled "true"
consul kv put ${NAMESPACE}/api/joox/timeout "10s"

# 网易云音乐API (Fallback 2)
consul kv put ${NAMESPACE}/api/netease/base_url "https://music.163.com/api"
consul kv put ${NAMESPACE}/api/netease/enabled "true"
consul kv put ${NAMESPACE}/api/netease/timeout "10s"

# 酷狗音乐API (Fallback 3)
consul kv put ${NAMESPACE}/api/kugou/base_url "https://www.kugou.com/yy"
consul kv put ${NAMESPACE}/api/kugou/enabled "true"
consul kv put ${NAMESPACE}/api/kugou/timeout "10s"

echo "✓ API configuration set"
echo ""

# ===== 短信配置 =====
echo "Setting SMS configuration..."

# 阿里云短信
consul kv put ${NAMESPACE}/sms/aliyun/access_key "your_aliyun_access_key"
consul kv put ${NAMESPACE}/sms/aliyun/secret_key "your_aliyun_secret_key"
consul kv put ${NAMESPACE}/sms/aliyun/sign_name "Listen Stream"
consul kv put ${NAMESPACE}/sms/aliyun/template_code "SMS_123456789"
consul kv put ${NAMESPACE}/sms/aliyun/enabled "true"

# 腾讯云短信 (Fallback 1)
consul kv put ${NAMESPACE}/sms/tencent/app_id "your_tencent_app_id"
consul kv put ${NAMESPACE}/sms/tencent/app_key "your_tencent_app_key"
consul kv put ${NAMESPACE}/sms/tencent/sign_name "Listen Stream"
consul kv put ${NAMESPACE}/sms/tencent/template_id "123456"
consul kv put ${NAMESPACE}/sms/tencent/enabled "false"

# Twilio短信 (Fallback 2)
consul kv put ${NAMESPACE}/sms/twilio/account_sid "your_twilio_account_sid"
consul kv put ${NAMESPACE}/sms/twilio/auth_token "your_twilio_auth_token"
consul kv put ${NAMESPACE}/sms/twilio/from_number "+1234567890"
consul kv put ${NAMESPACE}/sms/twilio/enabled "false"

# SMS提供商优先级
consul kv put ${NAMESPACE}/sms/provider_priority '["aliyun","tencent","twilio"]'

echo "✓ SMS configuration set"
echo ""

# ===== 功能开关 =====
echo "Setting feature flags..."

consul kv put ${NAMESPACE}/features/token_ip_binding "false"        # Token绑定IP(严格模式)
consul kv put ${NAMESPACE}/features/device_fingerprint "true"       # 设备指纹检测
consul kv put ${NAMESPACE}/features/strict_mode "false"             # 严格模式
consul kv put ${NAMESPACE}/features/2fa_required "false"            # 管理员强制2FA
consul kv put ${NAMESPACE}/features/offline_message "true"          # 离线消息队列

echo "✓ Feature flags set"
echo ""

# ===== 缓存配置 =====
echo "Setting cache configuration..."

# 三级缓存配置
consul kv put ${NAMESPACE}/cache/l1_memory_max_entries "1000"
consul kv put ${NAMESPACE}/cache/l1_memory_ttl "300"                # 5分钟
consul kv put ${NAMESPACE}/cache/l2_redis_ttl_default "3600"        # 1小时
consul kv put ${NAMESPACE}/cache/l3_stale_enabled "true"

# 缓存预热列表
consul kv put ${NAMESPACE}/cache/warmup_keys '["banner","hot_playlists"]'

echo "✓ Cache configuration set"
echo ""

# ===== 限流配置 =====
echo "Setting rate limit configuration..."

consul kv put ${NAMESPACE}/ratelimit/global_qps "10000"             # 全局QPS限制
consul kv put ${NAMESPACE}/ratelimit/per_user_qps "100"             # 单用户QPS
consul kv put ${NAMESPACE}/ratelimit/per_ip_qps "200"               # 单IP QPS
consul kv put ${NAMESPACE}/ratelimit/upstream_qq_music_qps "20"     # 上游限流

echo "✓ Rate limit configuration set"
echo ""

# ===== WebSocket配置 =====
echo "Setting WebSocket configuration..."

consul kv put ${NAMESPACE}/websocket/max_connections "10000"        # 最大连接数
consul kv put ${NAMESPACE}/websocket/heartbeat_interval "30"        # 心跳间隔(秒)
consul kv put ${NAMESPACE}/websocket/heartbeat_timeout "60"         # 心跳超时(秒)
consul kv put ${NAMESPACE}/websocket/message_queue_size "100"       # 每连接消息队列大小

echo "✓ WebSocket configuration set"
echo ""

# ===== 熔断器配置 =====
echo "Setting circuit breaker configuration..."

consul kv put ${NAMESPACE}/circuitbreaker/max_requests "5"          # 熔断前最大请求数
consul kv put ${NAMESPACE}/circuitbreaker/interval "10"             # 统计时间窗口(秒)
consul kv put ${NAMESPACE}/circuitbreaker/timeout "30"              # 半开状态超时(秒)
consul kv put ${NAMESPACE}/circuitbreaker/failure_threshold "5"     # 触发熔断的失败次数

echo "✓ Circuit breaker configuration set"
echo ""

echo "=========================================="
echo "✓ Consul KV initialization completed!"
echo "=========================================="
echo ""
echo "View all keys:"
echo "  consul kv get -recurse ${NAMESPACE}/"
echo ""
echo "Export for backup:"
echo "  consul kv export ${NAMESPACE}/ > backup.json"
echo ""
echo "Import from backup:"
echo "  consul kv import @backup.json"
