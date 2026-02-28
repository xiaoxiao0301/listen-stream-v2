package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/jwt"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// AuthConfig JWT认证配置
type AuthConfig struct {
	JWTSecret     string
	RequiredAuth  bool // 是否必须认证
	CheckIPBinding bool // 是否检查IP绑定
}

// Auth JWT认证中间件
func Auth(config AuthConfig, log logger.Logger) gin.HandlerFunc {
	jwtManager := jwt.NewManager(&jwt.Config{
		Secret: config.JWTSecret,
	})

	return func(c *gin.Context) {
		// 从Header获取Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			if config.RequiredAuth {
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    401,
					"message": "Missing authorization header",
				})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		// 解析Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 验证Token
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			log.WithFields(
				logger.String("request_id", GetRequestID(c)),
				logger.String("error", err.Error()),
			).Warn("JWT validation failed")

			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// 检查IP绑定（如果启用）
		if config.CheckIPBinding && claims.ClientIP != "" {
			clientIP := c.ClientIP()
			if clientIP != claims.ClientIP {
				log.WithFields(
					logger.String("request_id", GetRequestID(c)),
					logger.String("user_id", claims.UserID),
					logger.String("token_ip", claims.ClientIP),
					logger.String("request_ip", clientIP),
				).Warn("IP binding check failed")

				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    401,
					"message": "IP address mismatch",
				})
				c.Abort()
				return
			}
		}

		// 将用户信息存储到上下文
		c.Set("user_id", claims.UserID)
		c.Set("device_id", claims.DeviceID)
		c.Set("token_version", claims.TokenVersion)

		c.Next()
	}
}

// OptionalAuth 可选认证中间件（不强制要求Token）
func OptionalAuth(jwtSecret string, log logger.Logger) gin.HandlerFunc {
	return Auth(AuthConfig{
		JWTSecret:      jwtSecret,
		RequiredAuth:   false,
		CheckIPBinding: false,
	}, log)
}

// RequiredAuth 必须认证中间件
func RequiredAuth(jwtSecret string, log logger.Logger) gin.HandlerFunc {
	return Auth(AuthConfig{
		JWTSecret:      jwtSecret,
		RequiredAuth:   true,
		CheckIPBinding: false,
	}, log)
}

// StrictAuth 严格认证中间件（检查IP绑定）
func StrictAuth(jwtSecret string, log logger.Logger) gin.HandlerFunc {
	return Auth(AuthConfig{
		JWTSecret:      jwtSecret,
		RequiredAuth:   true,
		CheckIPBinding: true,
	}, log)
}
