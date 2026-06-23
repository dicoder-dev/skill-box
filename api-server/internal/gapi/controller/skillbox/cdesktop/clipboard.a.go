// clipboard.a.go - /api/desktop/clipboard/* 端点。
//
// 桌面端剪贴板走 OS API(macOS NSPasteboard),由 desktop 包封装后通过 hooks 注入。
// 跨 module 边界,cdesktop 不直接 import desktop。

package cdesktop

import (
	"ginp-api/pkg/ginp"

	"github.com/gin-gonic/gin"
)

// RequestSetClipboardText 写入剪贴板文本。
type RequestSetClipboardText struct {
	Text string `json:"text"`
}

// RequestEmptyClipboard 占位无入参(handler 只需要 c)。
type RequestEmptyClipboard struct{}

// GetClipboardText GET /api/desktop/clipboard/text
func GetClipboardText(c *ginp.ContextPlus, _ *RequestEmptyClipboard) {
	h := hooks()
	if h.ClipboardText == nil {
		c.JSON(501, gin.H{"error": "clipboard read not available"})
		return
	}
	text, err := h.ClipboardText()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"text": text})
}

// PutClipboardText PUT /api/desktop/clipboard/text { text }
func PutClipboardText(c *ginp.ContextPlus, req *RequestSetClipboardText) {
	h := hooks()
	if h.SetClipboardText == nil {
		c.JSON(501, gin.H{"error": "clipboard write not available"})
		return
	}
	if err := h.SetClipboardText(req.Text); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/clipboard/text", HttpType: ginp.HttpGet,
		Handler: ginp.BindParamsHandler(GetClipboardText, &RequestEmptyClipboard{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.clipboard.text.get", Description: "读取剪贴板文本", RequestParams: RequestEmptyClipboard{}},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/clipboard/text", HttpType: ginp.HttpPut,
		Handler: ginp.BindParamsHandler(PutClipboardText, &RequestSetClipboardText{}),
		Swagger: &ginp.SwaggerInfo{Title: "desktop.clipboard.text.set", Description: "写入剪贴板文本", RequestParams: RequestSetClipboardText{}},
	})
}