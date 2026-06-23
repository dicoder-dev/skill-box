// window.a.go - /api/desktop/window/* 端点。
//
// 桌面端窗口控制由 desktop.WindowManager 封装,通过 hooks 注入回调。
// 注意:windows 操作通常走主线程(macOS NSWindow 要求),hook 内部应自行处理线程切换。

package cdesktop

import (
	"ginp-api/internal/gapi/controller/skillbox/cdesktop/hooks"
	"ginp-api/pkg/ginp"

	"github.com/gin-gonic/gin"
)

// RequestWindowEmpty 占位无入参。
type RequestWindowEmpty struct{}

// PostWindowShow POST /api/desktop/window/show
func PostWindowShow(c *ginp.ContextPlus, _ *RequestWindowEmpty) {
	h := hooks.Get()
	if h.WindowShow == nil {
		c.JSON(501, gin.H{"error": "window.show not available"})
		return
	}
	h.WindowShow()
	c.JSON(200, gin.H{"ok": true})
}

// PostWindowToggleAlwaysOnTop POST /api/desktop/window/toggle-always-on-top
// 返回切换后的状态(true = 当前置顶)。
func PostWindowToggleAlwaysOnTop(c *ginp.ContextPlus, _ *RequestWindowEmpty) {
	h := hooks.Get()
	if h.WindowToggleAlwaysOnTop == nil {
		c.JSON(501, gin.H{"error": "window.toggleAlwaysOnTop not available"})
		return
	}
	state := h.WindowToggleAlwaysOnTop()
	c.JSON(200, gin.H{"on_top": state})
}

// PostWindowToggleMaximise POST /api/desktop/window/toggle-maximise
func PostWindowToggleMaximise(c *ginp.ContextPlus, _ *RequestWindowEmpty) {
	h := hooks.Get()
	if h.WindowToggleMaximise == nil {
		c.JSON(501, gin.H{"error": "window.toggleMaximise not available"})
		return
	}
	h.WindowToggleMaximise()
	c.JSON(200, gin.H{"ok": true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/window/show", HttpType: ginp.HttpPost,
		Handler: ginp.BindParamsHandler(PostWindowShow, &RequestWindowEmpty{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.window.show", Description: "主窗口显示", RequestParams: RequestWindowEmpty{}},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/window/toggle-always-on-top", HttpType: ginp.HttpPost,
		Handler: ginp.BindParamsHandler(PostWindowToggleAlwaysOnTop, &RequestWindowEmpty{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.window.toggleAlwaysOnTop", Description: "切换窗口置顶,返回切换后状态", RequestParams: RequestWindowEmpty{}},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/window/toggle-maximise", HttpType: ginp.HttpPost,
		Handler: ginp.BindParamsHandler(PostWindowToggleMaximise, &RequestWindowEmpty{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.window.toggleMaximise", Description: "切换窗口最大化", RequestParams: RequestWindowEmpty{}},
	})
}