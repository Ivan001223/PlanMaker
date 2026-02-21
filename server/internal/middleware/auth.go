package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kaoyan/server/internal/pkg/response"
)

// APIKey 内部 API 密钥 (通过环境变量 INTERNAL_API_KEY 设置)
var internalAPIKey string

// SetInternalAPIKey 设置内部 API 密钥
func SetInternalAPIKey(key string) {
	internalAPIKey = key
}

// InternalAPIMiddleware 内部 API 认证中间件
func InternalAPIMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果未配置 API Key，则拒绝所有请求
		if internalAPIKey == "" {
			response.Forbidden(c, "内部API未配置密钥")
			c.Abort()
			return
		}

		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			// 尝试从 query 参数获取
			apiKey = c.Query("api_key")
		}

		if apiKey == "" || apiKey != internalAPIKey {
			response.Forbidden(c, "API密钥无效")
			c.Abort()
			return
		}

		c.Next()
	}
}

// JWTClaims JWT 自定义 Claims
type JWTClaims struct {
	UserID uint64 `json:"user_id"`
	OpenID string `json:"open_id"`
	jwt.RegisteredClaims
}

var jwtSecret []byte

// InitJWT 初始化 JWT 密钥
func InitJWT(secret string) {
	jwtSecret = []byte(secret)
}

// GenerateToken 生成 JWT Token
func GenerateToken(userID uint64, openID string, expireHours int) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		OpenID: openID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "kaoyan-server",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// AuthMiddleware JWT 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims := &JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			response.Unauthorized(c, "token无效或已过期")
			c.Abort()
			return
		}

		// 将用户信息存入 Context
		c.Set("user_id", claims.UserID)
		c.Set("open_id", claims.OpenID)
		c.Next()
	}
}

// GetUserID 从 Context 中获取用户ID
func GetUserID(c *gin.Context) uint64 {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	return userID.(uint64)
}
