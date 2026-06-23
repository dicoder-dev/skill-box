// fs.a.go - /api/desktop/fs/* 端点。
//
// 提供桌面端的本地文件能力:
//   - POST /api/desktop/fs/read-text { path }    读文件文本(用于 SKILL.md 标题展示)
//   - POST /api/desktop/fs/reveal   { path }    在系统文件管理器中显示该路径
//
// 实现位于 internal/fsutil,跨 cdesktop / desktop 包复用,避免循环依赖。
// 安全性:大小上限 1 MB + path 由前端传,只对 adapter 报出来的 source_path 触发,
// 不做任意路径读取,风险面较小。

package cdesktop

import (
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/controller/skillbox/cdesktop/hooks"
	"ginp-api/pkg/ginp"
	"skill-box/pkg/fsutil"
)

// RequestFsReadText { path }
type RequestFsReadText struct {
	Path string `json:"path"`
}

// RespondFsReadText { content }
type RespondFsReadText struct {
	Content string `json:"content"`
}

// PostFsReadText POST /api/desktop/fs/read-text
func PostFsReadText(c *ginp.ContextPlus, req *RequestFsReadText) {
	if strings.TrimSpace(req.Path) == "" {
		c.JSON(400, gin.H{"error": "missing path"})
		return
	}
	h := hooks.Get()
	if h.FsReadText == nil {
		// Web 端没桌面 hook,直接走 fsutil 读本地文件
		content, err := fsutil.ReadText(req.Path)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, RespondFsReadText{Content: content})
		return
	}
	content, err := h.FsReadText(req.Path)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, RespondFsReadText{Content: content})
}

// RequestFsReveal { path }
type RequestFsReveal struct {
	Path string `json:"path"`
}

// PostFsReveal POST /api/desktop/fs/reveal
func PostFsReveal(c *ginp.ContextPlus, req *RequestFsReveal) {
	if strings.TrimSpace(req.Path) == "" {
		c.JSON(400, gin.H{"error": "missing path"})
		return
	}
	h := hooks.Get()
	if h.FsReveal == nil {
		// 兜底:返回 file:// 父目录 URL,前端可以走 openExternal
		parent := filepath.Dir(req.Path)
		if abs, err := filepath.Abs(parent); err == nil {
			c.JSON(501, gin.H{"error": "fs.reveal not available", "fallback_url": "file://" + abs})
			return
		}
		c.JSON(501, gin.H{"error": "fs.reveal not available"})
		return
	}
	if err := h.FsReveal(req.Path); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/fs/read-text", HttpType: ginp.HttpPost,
		Handler: ginp.BindParamsHandler(PostFsReadText, &RequestFsReadText{}),
		Swagger: &ginp.SwaggerInfo{
			Title:         "desktop.fs.readText",
			Description:   "读本地文件文本(上限 1 MB),用于 SKILL.md 标题展示等",
			RequestParams: RequestFsReadText{},
		},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/fs/reveal", HttpType: ginp.HttpPost,
		Handler: ginp.BindParamsHandler(PostFsReveal, &RequestFsReveal{}),
		Swagger: &ginp.SwaggerInfo{
			Title:         "desktop.fs.reveal",
			Description:   "在系统文件管理器中显示给定路径(文件会高亮,目录会直接打开)",
			RequestParams: RequestFsReveal{},
		},
	})
}
