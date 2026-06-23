// shortcut.a.go - /api/desktop/shortcut/* 端点。
//
// 全局快捷键由 desktop.ShortcutManager 封装(macOS Carbon / Windows RegisterHotKey)。
// hook 内部已处理线程切换 + 失败 error,handler 只透传。

package cdesktop

import (
	"ginp-api/pkg/ginp"

	"github.com/gin-gonic/gin"
)

// RequestShortcutCombo 全局快捷键 combo(入参)。
type RequestShortcutCombo struct {
	Combo string `json:"combo"`
}

// PostShortcutRegister POST /api/desktop/shortcut/register { combo }
func PostShortcutRegister(c *ginp.ContextPlus, req *RequestShortcutCombo) {
	h := hooks()
	if h.ShortcutRegister == nil {
		c.JSON(501, gin.H{"error": "shortcut.register not available"})
		return
	}
	if req.Combo == "" {
		c.JSON(400, gin.H{"error": "missing combo"})
		return
	}
	if err := h.ShortcutRegister(req.Combo); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

// PostShortcutUnregister POST /api/desktop/shortcut/unregister { combo }
func PostShortcutUnregister(c *ginp.ContextPlus, req *RequestShortcutCombo) {
	h := hooks()
	if h.ShortcutUnregister == nil {
		c.JSON(501, gin.H{"error": "shortcut.unregister not available"})
		return
	}
	if req.Combo == "" {
		c.JSON(400, gin.H{"error": "missing combo"})
		return
	}
	if err := h.ShortcutUnregister(req.Combo); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

// GetShortcutList GET /api/desktop/shortcut/list
func GetShortcutList(c *ginp.ContextPlus, _ *RequestShortcutCombo) {
	h := hooks()
	if h.ShortcutList == nil {
		c.JSON(200, gin.H{"combos": []string{}})
		return
	}
	c.JSON(200, gin.H{"combos": h.ShortcutList()})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/shortcut/register", HttpType: ginp.HttpPost,
		Handler: ginp.BindParamsHandler(PostShortcutRegister, &RequestShortcutCombo{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.shortcut.register", Description: "注册一个全局快捷键", RequestParams: RequestShortcutCombo{}},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/shortcut/unregister", HttpType: ginp.HttpPost,
		Handler: ginp.BindParamsHandler(PostShortcutUnregister, &RequestShortcutCombo{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.shortcut.unregister", Description: "注销一个全局快捷键", RequestParams: RequestShortcutCombo{}},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/shortcut/list", HttpType: ginp.HttpGet,
		Handler: ginp.BindParamsHandler(GetShortcutList, &RequestShortcutCombo{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.shortcut.list", Description: "列出已注册的全部 combo", RequestParams: RequestShortcutCombo{}},
	})
}