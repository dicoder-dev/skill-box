package ginp

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// ANSI 颜色代码
const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorReset  = "\033[0m"
)

// LoggingMiddleware 请求日志中间件
// 记录每个请求的方法、路径、状态码和耗时
// 非200状态码使用红色字体打印
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()

		// 处理请求
		c.Next()

		// 记录响应信息
		statusCode := c.Writer.Status()
		duration := time.Since(startTime)

		if showLog {
			// 非200状态码使用红色字体打印
			if statusCode >= 400 {
				log.Printf("%s[%s] %s %s | Status: %d | Duration: %v%s",
					ColorRed,
					clientIP,
					method,
					path,
					statusCode,
					duration,
					ColorReset,
				)
			} else {
				log.Printf("[%s] %s %s | Status: %d | Duration: %v",
					clientIP,
					method,
					path,
					statusCode,
					duration,
				)
			}
		}
	}
}

// CORSMiddleware CORS 跨域中间件
// 配置跨域允许的方法、头部和来源
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RecoveryMiddleware 恢复中间件
// 捕获 panic 异常，返回 500 错误而不是崩溃
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				if showLog {
					log.Printf("[PANIC] %v", err)
				}
				c.JSON(500, gin.H{
					"code": codeFail,
					"msg":  "Internal server error",
				})
			}
		}()
		c.Next()
	}
}

// RequestIDMiddleware 请求 ID 中间件
// 为每个请求生成唯一的 ID，便于追踪日志
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetString("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
			c.Set("X-Request-ID", requestID)
		}
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}

// generateRequestID 生成唯一的请求 ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

// randomString 生成随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}
