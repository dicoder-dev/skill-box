// external.a.go - /api/desktop/open-external 端点。
//
// 用系统默认浏览器打开 URL。OS 层由 desktop 包通过 wails platform.BrowserOpenURL 实现,
// 此处由 OpenExternal hook 代理。

package cdesktop

import (
	"ginp-api/pkg/ginp"

	"github.com/gin-gonic/gin"
)

// RequestOpenExternal 入参 { url }。
type RequestOpenExternal struct {
	URL string `json:"url"`
}

// PostOpenExternal POST /api/desktop/open-external { url }
func PostOpenExternal(c *ginp.ContextPlus, req *RequestOpenExternal) {
	h := hooks()
	if h.OpenExternal == nil {
		c.JSON(501, gin.H{"error": "openExternal not available"})
		return
	}
	if req.URL == "" {
		c.JSON(400, gin.H{"error": "missing url"})
		return
	}
	if err := h.OpenExternal(req.URL); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/open-external", HttpType: ginp.HttpPost,
		Handler: ginp.BindParamsHandler(PostOpenExternal, &RequestOpenExternal{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.openExternal", Description: "用系统默认浏览器打开 URL", RequestParams: RequestOpenExternal{}},
	})
}