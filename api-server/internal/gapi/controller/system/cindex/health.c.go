package cindex

import (
	"net/http"
	"time"

	"ginp-api/pkg/ginp"

	"github.com/gin-gonic/gin"
)

// Health 健康检查，桌面端 Webview 与前端探测本地 server 端口是否就绪。
// 不参与 CORS、登录鉴权，是双部署形态下前端确认"后端活着"的唯一稳定锚点。
func Health(c *ginp.ContextPlus) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "ginp-api",
		"ts":      time.Now().Unix(),
	})
}
