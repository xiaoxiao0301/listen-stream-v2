package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserKey(t *testing.T) {
	key := UserKey("12345")
	assert.Equal(t, "ls:user:12345", key)
}

func TestUserTokenKey(t *testing.T) {
	key := UserTokenKey("12345")
	assert.Equal(t, "ls:user:12345:token_version", key)
}

func TestUserDeviceKey(t *testing.T) {
	key := UserDeviceKey("user123", "device456")
	assert.Equal(t, "ls:user:user123:device:device456", key)
}

func TestSessionKey(t *testing.T) {
	key := SessionKey("token_hash_123")
	assert.Equal(t, "ls:session:token_hash_123", key)
}

func TestRefreshTokenKey(t *testing.T) {
	key := RefreshTokenKey("refresh_hash_123")
	assert.Equal(t, "ls:refresh:refresh_hash_123", key)
}

func TestSMSVerificationKey(t *testing.T) {
	key := SMSVerificationKey("+1234567890")
	assert.Equal(t, "ls:sms:verify:+1234567890", key)
}

func TestSMSRateLimitKey(t *testing.T) {
	key := SMSRateLimitKey("+1234567890", "daily")
	assert.Equal(t, "ls:sms:limit:+1234567890:daily", key)
}

func TestCacheKey(t *testing.T) {
	key := CacheKey("song", "12345")
	assert.Equal(t, "ls:cache:song:12345", key)
}

func TestKeyBuilder(t *testing.T) {
	kb := NewKeyBuilder()
	key := kb.Entity("user").ID("123").Field("profile").Build()
	assert.Equal(t, "ls:user:123:profile", key)
}

func TestKeyBuilder_MultipleFields(t *testing.T) {
	kb := NewKeyBuilder()
	key := kb.Entity("playlist").ID("456").Field("songs").Field("count").Build()
	assert.Equal(t, "ls:playlist:456:songs:count", key)
}

func TestKeyBuilder_OnlyEntity(t *testing.T) {
	kb := NewKeyBuilder()
	key := kb.Entity("stats").Build()
	assert.Equal(t, "ls:stats", key)
}
