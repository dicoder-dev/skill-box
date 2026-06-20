package router

import (
	"context"
	"ginp-api/configs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	// KeyUser 是 Gin 框架中用于在请求上下文中存储 JWT 令牌的键名。当 JWT 验证成功后，解析出的令牌会被存储到 Gin 的上下文中，后续的处理器可以通过这个键名获取到用户信息。
	//   token, exists := c.Get(KeyUser)  // 从上下文中获取用户令牌
	KeyUser = "jwt_user"
)

// ✅ 全局变量：JWKS URL
var JwksURL string

// ✅ 全局变量：固定的 KeySet （因为自建用户中心JWKS是固定的）
var keySet jwk.Set

// InitJWK 初始化 JWKS 加载器
func InitJWK() error {
	// 设置全局变量
	JwksURL = configs.SystemUserCenterUrl() + "/.well-known/jwks.json"

	ctx := context.Background()

	// 一次性加载，因为自建用户中心的JWKS是固定的
	set, err := jwk.Fetch(ctx, JwksURL)
	if err != nil {
		return err
	}
	if set.Len() == 0 {
		return context.DeadlineExceeded
	}

	// 保存到全局变量，避免重复请求
	keySet = set

	return nil
}

// AuthUserCenterMiddleware 自建用户中心JWT 鉴权中间件
func AuthUserCenterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前请求路径
		currentPath := c.Request.URL.Path

		// 查找当前路径对应的路由项
		currentRouter := findRouterByPath(currentPath)

		// 如果找不到路由项或不需要登录，则跳过鉴权
		if currentRouter == nil || !currentRouter.NeedLogin {
			c.Next()
			return
		}

		// 需要登录，进行JWT鉴权
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// ✅ 使用全局 keySet
		if keySet == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "public keys not initialized",
			})
			c.Abort()
			return
		}

		token, err := jwt.Parse([]byte(tokenStr), jwt.WithKeySet(keySet))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token: " + err.Error()})
			c.Abort()
			return
		}

		c.Set(KeyUser, token)
		c.Next()
	}
}

// OptionalAuthMiddleware 可选鉴权
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if keySet != nil {
				token, err := jwt.Parse([]byte(tokenStr), jwt.WithKeySet(keySet))
				if err == nil {
					c.Set(KeyUser, token)
				}
			}
		}
		c.Next()
	}
}
