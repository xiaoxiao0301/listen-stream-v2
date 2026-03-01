#!/bin/bash
# Initialize Consul KV Store with default values for Listen Stream
# Usage: ./init-kv.sh [CONSUL_HTTP_ADDR]

set -e

CONSUL_ADDR="${1:-http://localhost:8500}"
CONSUL_KV_PREFIX="listen-stream"

echo "==> Initializing Consul KV Store at ${CONSUL_ADDR}"
echo "==> Prefix: ${CONSUL_KV_PREFIX}"
echo ""

# Wait for Consul to be ready
MAX_TRIES=30
TRIES=0
until curl -sf "${CONSUL_ADDR}/v1/status/leader" > /dev/null 2>&1; do
    TRIES=$((TRIES + 1))
    if [ $TRIES -ge $MAX_TRIES ]; then
        echo "ERROR: Consul not ready after ${MAX_TRIES} attempts"
        exit 1
    fi
    echo "Waiting for Consul... (attempt ${TRIES}/${MAX_TRIES})"
    sleep 2
done

echo "==> Consul is ready"
echo ""

# Helper function to set KV
kv_put() {
    local key="$1"
    local value="$2"
    curl -sf -X PUT -d "${value}" "${CONSUL_ADDR}/v1/kv/${CONSUL_KV_PREFIX}/${key}" > /dev/null
    echo "  SET: ${CONSUL_KV_PREFIX}/${key}"
}

# ============================================================
# 1. Common / JWT Configuration
# ============================================================
echo "==> [1/6] Setting common/JWT configuration..."
kv_put "common/jwt_secret"  "listen-stream-dev-secret-change-in-production"
kv_put "common/jwt_version" "1"
kv_put "common/aes_key"     "listen-stream-aes-256-key-32bytes!!"

# ============================================================
# 2. QQ Music API
# ============================================================
echo "==> [2/6] Setting QQ Music API configuration..."
kv_put "api/qq_music/base_url"  "https://u.y.qq.com"
kv_put "api/qq_music/api_key"   "dev-qq-music-api-key"
kv_put "api/qq_music/timeout"   "10s"
kv_put "api/qq_music/max_retry" "3"

# Joox
kv_put "api/joox/base_url"  "https://api.joox.com"
kv_put "api/joox/cookie"    "dev-joox-cookie"
kv_put "api/joox/timeout"   "10s"

# NetEase
kv_put "api/netease/base_url" "https://music.163.com"
kv_put "api/netease/timeout"  "10s"

# Kugou
kv_put "api/kugou/base_url" "https://m.kugou.com"
kv_put "api/kugou/timeout"  "10s"

# ============================================================
# 3. SMS Configuration
# ============================================================
echo "==> [3/6] Setting SMS configuration..."
kv_put "sms/aliyun/access_key_id"     "dev-aliyun-access-key-id"
kv_put "sms/aliyun/access_key_secret" "dev-aliyun-access-key-secret"
kv_put "sms/aliyun/sign_name"         "Listen Stream"
kv_put "sms/aliyun/template_code"     "SMS_000000"

kv_put "sms/tencent/secret_id"        "dev-tencent-secret-id"
kv_put "sms/tencent/secret_key"       "dev-tencent-secret-key"
kv_put "sms/tencent/app_id"           "dev-tencent-app-id"
kv_put "sms/tencent/sign_name"        "Listen Stream"

kv_put "sms/twilio/account_sid"       "dev-twilio-account-sid"
kv_put "sms/twilio/auth_token"        "dev-twilio-auth-token"
kv_put "sms/twilio/from_number"       "+1234567890"

# ============================================================
# 4. Feature Flags
# ============================================================
echo "==> [4/6] Setting feature flags..."
kv_put "features/token_ip_binding"    "false"
kv_put "features/device_fingerprint"  "true"
kv_put "features/l1_cache_enabled"    "true"
kv_put "features/fallback_chain"      "qq_music,joox,netease,kugou"
kv_put "features/sms_provider_order"  "aliyun,tencent,twilio"
kv_put "features/max_devices"         "5"
kv_put "features/max_play_history"    "500"

# ============================================================
# 5. Rate Limiter Configuration
# ============================================================
echo "==> [5/6] Setting rate limiter configuration..."
kv_put "rate_limit/global_rps"        "10000"
kv_put "rate_limit/user_rps"          "100"
kv_put "rate_limit/ip_rps"            "200"
kv_put "rate_limit/upstream_rps"      "20"
kv_put "rate_limit/sms_per_phone"     "5"
kv_put "rate_limit/sms_per_ip"        "20"

# ============================================================
# 6. Circuit Breaker Configuration
# ============================================================
echo "==> [6/6] Setting circuit breaker configuration..."
kv_put "breaker/max_requests"           "100"
kv_put "breaker/interval"               "60s"
kv_put "breaker/timeout"                "30s"
kv_put "breaker/consecutive_failures"   "5"

echo ""
echo "==> ✓ Consul KV initialized successfully!"
echo ""
echo "==> Verify with:"
echo "    curl ${CONSUL_ADDR}/v1/kv/${CONSUL_KV_PREFIX}/?recurse | jq ."
echo ""
echo "==> Or browse the UI at: ${CONSUL_ADDR}/ui"
