package cindex

import (
	"net/http"
	"time"

	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestHealth 健康检查请求参数(GET 无 body,保留结构体以对齐 BindParamsHandler 形态)
type RequestHealth struct {
}

// RespondHealth 健康检查响应结构
type RespondHealth struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Ts      int64  `json:"ts"`
}

// Health 健康检查,桌面端 Webview 与前端探测本地 server 端口是否就绪。
// 不参与 CORS、登录鉴权,是双部署形态下前端确认"后端活着"的唯一稳定锚点。
func Health(c *ginp.ContextPlus, requestParams *RequestHealth) {
	logger.Info("ok")
	c.JSON(http.StatusOK, RespondHealth{
		Status:  "ok",
		Service: "ginp-api",
		Ts:      time.Now().Unix(),
	})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/health",
		Handler:        ginp.BindParamsHandler(Health, RequestHealth{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "ok",
		Swagger: &ginp.SwaggerInfo{
			Title:         "health",
			Description:   "健康检查接口,用于前端确认后端服务存活",
			RequestParams: RequestHealth{},
		},
	})
}
