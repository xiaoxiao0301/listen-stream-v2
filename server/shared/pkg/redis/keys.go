package redis

import (
	"fmt"
	"strings"
)

// Key naming conventions for Redis keys.
// All keys follow the pattern: {namespace}:{entity}:{id}:{field}
//
// Example: "ls:user:123:token" for user 123's token

const (
	// Namespace prefix for all keys
	KeyNamespace = "ls" // Listen Stream
)

// KeyBuilder helps build Redis keys following naming conventions.
type KeyBuilder struct {
	parts []string
}

// NewKeyBuilder creates a new key builder.
func NewKeyBuilder() *KeyBuilder {
	return &KeyBuilder{
		parts: []string{KeyNamespace},
	}
}

// Entity adds an entity type to the key.
func (kb *KeyBuilder) Entity(entity string) *KeyBuilder {
	kb.parts = append(kb.parts, entity)
	return kb
}

// ID adds an ID to the key.
func (kb *KeyBuilder) ID(id string) *KeyBuilder {
	kb.parts = append(kb.parts, id)
	return kb
}

// Field adds a field name to the key.
func (kb *KeyBuilder) Field(field string) *KeyBuilder {
	kb.parts = append(kb.parts, field)
	return kb
}

// Build constructs the final key string.
func (kb *KeyBuilder) Build() string {
	return strings.Join(kb.parts, ":")
}

// Predefined key builders for common entities

// UserKey returns a key for user data.
// Example: ls:user:123
func UserKey(userID string) string {
	return fmt.Sprintf("%s:user:%s", KeyNamespace, userID)
}

// UserTokenKey returns a key for user JWT token version.
// Example: ls:user:123:token_version
func UserTokenKey(userID string) string {
	return fmt.Sprintf("%s:user:%s:token_version", KeyNamespace, userID)
}

// UserDeviceKey returns a key for a user's device.
// Example: ls:user:123:device:abc
func UserDeviceKey(userID, deviceID string) string {
	return fmt.Sprintf("%s:user:%s:device:%s", KeyNamespace, userID, deviceID)
}

// SessionKey returns a key for a user session.
// Example: ls:session:token_hash
func SessionKey(tokenHash string) string {
	return fmt.Sprintf("%s:session:%s", KeyNamespace, tokenHash)
}

// RefreshTokenKey returns a key for a refresh token.
// Example: ls:refresh:token_hash
func RefreshTokenKey(tokenHash string) string {
	return fmt.Sprintf("%s:refresh:%s", KeyNamespace, tokenHash)
}

// SMSVerificationKey returns a key for SMS verification code.
// Example: ls:sms:verify:+1234567890
func SMSVerificationKey(phone string) string {
	return fmt.Sprintf("%s:sms:verify:%s", KeyNamespace, phone)
}

// SMSRateLimitKey returns a key for SMS rate limiting.
// Example: ls:sms:limit:+1234567890:daily
func SMSRateLimitKey(phone, period string) string {
	return fmt.Sprintf("%s:sms:limit:%s:%s", KeyNamespace, phone, period)
}

// CacheKey returns a key for cached data.
// Example: ls:cache:song:123
func CacheKey(entity, id string) string {
	return fmt.Sprintf("%s:cache:%s:%s", KeyNamespace, entity, id)
}

// CachePattern returns a pattern for cache keys.
// Example: ls:cache:song:*
func CachePattern(entity string) string {
	return fmt.Sprintf("%s:cache:%s:*", KeyNamespace, entity)
}

// StaleCache Key returns a key for stale cache data (L3 cache).
// Example: ls:stale:song:123
func StaleCacheKey(entity, id string) string {
	return fmt.Sprintf("%s:stale:%s:%s", KeyNamespace, entity, id)
}

// RateLimitKey returns a key for rate limiting.
// Example: ls:ratelimit:ip:192.168.1.1:minute
func RateLimitKey(rtype, identifier, window string) string {
	return fmt.Sprintf("%s:ratelimit:%s:%s:%s", KeyNamespace, rtype, identifier, window)
}

// OfflineMessageKey returns a key for offline messages list.
// Example: ls:offline:user:123
func OfflineMessageKey(userID string) string {
	return fmt.Sprintf("%s:offline:user:%s", KeyNamespace, userID)
}

// ConnectionKey returns a key for WebSocket connection tracking.
// Example: ls:conn:user:123
func ConnectionKey(userID string) string {
	return fmt.Sprintf("%s:conn:user:%s", KeyNamespace, userID)
}

// PubSubChannel returns a channel name for pub/sub.
// Example: ls:pubsub:config:change
func PubSubChannel(topic string) string {
	return fmt.Sprintf("%s:pubsub:%s", KeyNamespace, topic)
}

// ConfigChangeChannel returns the channel for configuration changes.
func ConfigChangeChannel() string {
	return PubSubChannel("config:change")
}

// SyncEventChannel returns the channel for sync events.
// Example: ls:pubsub:sync:user:123
func SyncEventChannel(userID string) string {
	return PubSubChannel(fmt.Sprintf("sync:user:%s", userID))
}

// LockKey returns a key for distributed locks.
// Example: ls:lock:resource:123
func LockKey(resource string) string {
	return fmt.Sprintf("%s:lock:%s", KeyNamespace, resource)
}

// DailyStatsKey returns a key for daily statistics.
// Example: ls:stats:daily:2026-02-26:active_users
func DailyStatsKey(date, metric string) string {
	return fmt.Sprintf("%s:stats:daily:%s:%s", KeyNamespace, date, metric)
}

// RealtimeStatsKey returns a key for realtime statistics.
// Example: ls:stats:rt:active_connections
func RealtimeStatsKey(metric string) string {
	return fmt.Sprintf("%s:stats:rt:%s", KeyNamespace, metric)
}

// PlaylistKey returns a key for playlist data.
// Example: ls:playlist:123
func PlaylistKey(playlistID string) string {
	return fmt.Sprintf("%s:playlist:%s", KeyNamespace, playlistID)
}

// FavoriteSetKey returns a key for a user's favorite songs set.
// Example: ls:favorites:user:123
func FavoriteSetKey(userID string) string {
	return fmt.Sprintf("%s:favorites:user:%s", KeyNamespace, userID)
}

// PlayHistoryKey returns a key for play history list.
// Example: ls:history:user:123
func PlayHistoryKey(userID string) string {
	return fmt.Sprintf("%s:history:user:%s", KeyNamespace, userID)
}

// CircuitBreakerKey returns a key for circuit breaker state.
// Example: ls:breaker:qq_music
func CircuitBreakerKey(service string) string {
	return fmt.Sprintf("%s:breaker:%s", KeyNamespace, service)
}

// APIMetricsKey returns a key for API metrics.
// Example: ls:metrics:api:qq_music:requests
func APIMetricsKey(api, metric string) string {
	return fmt.Sprintf("%s:metrics:api:%s:%s", KeyNamespace, api, metric)
}

// KeyTTL defines common TTL durations for different key types.
var KeyTTL = struct {
	Session          int // 1 hour
	RefreshToken     int // 7 days
	SMSVerification  int // 5 minutes
	SMSRateLimit     int // 24 hours
	CacheShort       int // 5 minutes
	CacheMedium      int // 1 hour
	CacheLong        int // 24 hours
	StaleCache       int // 7 days
	RateLimitMinute  int // 1 minute
	RateLimitHour    int // 1 hour
	OfflineMessage   int // 7 days
	Connection       int // 1 hour (with refresh)
	Lock             int // 30 seconds
	DailyStats       int // 90 days
	RealtimeStats    int // 1 hour
	CircuitBreaker   int // 1 hour
}{
	Session:         3600,
	RefreshToken:    604800,
	SMSVerification: 300,
	SMSRateLimit:    86400,
	CacheShort:      300,
	CacheMedium:     3600,
	CacheLong:       86400,
	StaleCache:      604800,
	RateLimitMinute: 60,
	RateLimitHour:   3600,
	OfflineMessage:  604800,
	Connection:      3600,
	Lock:            30,
	DailyStats:      7776000,
	RealtimeStats:   3600,
	CircuitBreaker:  3600,
}
