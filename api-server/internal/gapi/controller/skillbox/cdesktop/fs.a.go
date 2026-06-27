// fs.a.go - /api/desktop/fs/* 端点。
//
// 提供桌面端的本地文件能力:
//   - POST /api/desktop/fs/read-text      { path }                读文件文本(用于 SKILL.md 标题展示)
//   - POST /api/desktop/fs/reveal         { path }                在系统文件管理器中显示该路径
//   - POST /api/desktop/fs/pick-folder    {}                      弹系统对话框选目录,返回绝对路径
//   - POST /api/desktop/fs/inspect-project { path }                从目录路径推断项目 name/alias
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

// RespondFsPickFolder { path }。path 为空表示用户取消选择。
type RespondFsPickFolder struct {
	Path string `json:"path"`
}

// PostFsPickFolder POST /api/desktop/fs/pick-folder
//
// 弹系统文件夹选择对话框,返回用户选中的绝对路径。取消时返 200 + path="",
// 不要当作 error 处理。
func PostFsPickFolder(c *ginp.ContextPlus) {
	h := hooks.Get()
	if h.FsPickFolder == nil {
		// Web 端无桌面 hook,前端走降级(隐藏按钮或转 input[webkitdirectory])
		c.JSON(501, gin.H{"error": "fs.pickFolder not available in web mode"})
		return
	}
	path, err := h.FsPickFolder()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, RespondFsPickFolder{Path: path})
}

// RequestFsInspectProject { path }
type RequestFsInspectProject struct {
	Path string `json:"path"`
}

// RespondFsInspectProject { name, alias }。path 不合法时 400。
type RespondFsInspectProject struct {
	Name  string `json:"name"`
	Alias string `json:"alias"`
}

// PostFsInspectProject POST /api/desktop/fs/inspect-project
//
// 从给定的目录路径推断"项目元信息",供"导入项目"流程预填表单。
// Web 端和桌面端都用同一个 fsutil 实现,行为完全一致。
func PostFsInspectProject(c *ginp.ContextPlus, req *RequestFsInspectProject) {
	if strings.TrimSpace(req.Path) == "" {
		c.JSON(400, gin.H{"error": "missing path"})
		return
	}
	hint, err := fsutil.InspectProject(req.Path)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, RespondFsInspectProject{Name: hint.Name, Alias: hint.Alias})
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
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/fs/pick-folder", HttpType: ginp.HttpPost,
		Handler: ginp.BindParamsHandler(PostFsPickFolder, &struct{}{}),
		Swagger: &ginp.SwaggerInfo{
			Title:       "desktop.fs.pickFolder",
			Description: "弹系统文件夹选择对话框;取消时返 { path: \"\" }。Web 端返 501,前端降级。",
		},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path: "/api/desktop/fs/inspect-project", HttpType: ginp.HttpPost,
		Handler: ginp.BindParamsHandler(PostFsInspectProject, &RequestFsInspectProject{}),
		Swagger: &ginp.SwaggerInfo{
			Title:         "desktop.fs.inspectProject",
			Description:   "从目录路径推断项目 name / alias(name 取 basename,alias 走 slugify)",
			RequestParams: RequestFsInspectProject{},
		},
	})
}
