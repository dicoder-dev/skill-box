// notify.a.go - /api/desktop/notify/* 端点。
//
// 桌面端通知走 wails v3 的 darwin UNUserNotificationCenter(由 desktop.Notifier 封装)。
// 后端 cdesktop 不直接 import wails(跨 module 边界),通过 hooks() 调到 desktop 注入的
// Notify / NotifyHasPermission / NotifyRequestAuthorization 三个回调。
//
// 同步语义:HTTP handler 在自己的 goroutine 里跑,wails notifier.SendNotification 也走
// CGO delegate,失败由 hook 返回 error,handler 透传给前端。

package cdesktop

import (
	"ginp-api/internal/gapi/controller/skillbox/cdesktop/hooks"
	"ginp-api/pkg/ginp"

	"github.com/gin-gonic/gin"
)

// RequestNotifyShow 发送一条系统通知。
type RequestNotifyShow struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// RequestNotifyPermission 占位无入参。
type RequestNotifyPermission struct{}

// PostNotifyShow POST /api/desktop/notify/show
func PostNotifyShow(c *ginp.ContextPlus, req *RequestNotifyShow) {
	h := hooks.Get()
	if h.Notify == nil {
		c.JSON(501, gin.H{"error": "notify: not available in current deployment (web/headless)"})
		return
	}
	if req.Title == "" {
		c.JSON(400, gin.H{"error": "missing title"})
		return
	}
	if err := h.Notify(req.ID, req.Title, req.Body); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

// GetNotifyPermission GET /api/desktop/notify/permission
func GetNotifyPermission(c *ginp.ContextPlus, _ *RequestNotifyPermission) {
	h := hooks.Get()
	if h.NotifyHasPermission == nil {
		c.JSON(200, gin.H{"granted": false, "available": false})
		return
	}
	c.JSON(200, gin.H{"granted": h.NotifyHasPermission()})
}

// PostNotifyPermissionRequest POST /api/desktop/notify/permission/request
// 触发系统弹授权窗(macOS 首次启动)。同步等用户响应或超时。
func PostNotifyPermissionRequest(c *ginp.ContextPlus, _ *RequestNotifyPermission) {
	h := hooks.Get()
	if h.NotifyRequestAuthorization == nil {
		c.JSON(501, gin.H{"error": "notify authorization not available"})
		return
	}
	ok, err := h.NotifyRequestAuthorization()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"granted": ok})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/notify/show", HttpType: ginp.HttpPost,
		Handler: ginp.BindParamsHandler(PostNotifyShow, &RequestNotifyShow{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.notify.show", Description: "发送一条系统通知", RequestParams: RequestNotifyShow{}},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/notify/permission", HttpType: ginp.HttpGet,
		Handler: ginp.BindParamsHandler(GetNotifyPermission, &RequestNotifyPermission{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.notify.permission", Description: "查询当前通知授权状态", RequestParams: RequestNotifyPermission{}},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/notify/permission/request", HttpType: ginp.HttpPost,
		Handler: ginp.BindParamsHandler(PostNotifyPermissionRequest, &RequestNotifyPermission{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.notify.permission.request", Description: "触发系统弹授权窗", RequestParams: RequestNotifyPermission{}},
	})
}