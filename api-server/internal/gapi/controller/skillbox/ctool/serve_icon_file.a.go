// Package ctool - serve_icon_file.a.go
// GET /api/files/tool-icons/*filename
//
// 静态服务 ~/.skill-box/tool-icons/ 下的图片文件(用户上传 + seed 嵌入的)。
// 防穿越:文件名必须经过 toolicon.ValidIconFileName 校验。
//
// 这里用 ginp.RouterAppend 注册 GET 路由(参数 filename),而不是 FileServer,
// 因为 FileServer 无法做精细化校验。
package ctool

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/tool/toolicon"
	"ginp-api/pkg/ginp"
)

// ServeIconFile GET /api/files/tool-icons/&lt;filename&gt;
//
// 路由段绑定 :filename(不是 gin 的 *filepath),之所以不用 * 通配符:
//   - gin 的 *filepath 会与 SPA fallback 的通配顺序冲突,实际请求
//     落不到这里;用 :param 段绑定是没歧义的精确子路径匹配。
//   - 调用方 fetch URL 拼成 "/api/files/tool-icons/<basename>" 不变。
//
// 静态文件服务 ~/.skill-box/tool-icons/&lt;name&gt;,文件名必须经过
// toolicon.ValidIconFileName + filepath.Base 二重校验防穿越。
func ServeIconFile(c *ginp.ContextPlus) {
	name := strings.TrimSpace(c.Param("filename"))
	if name == "" {
		name = strings.TrimPrefix(c.Param("filepath"), "/")
	}
	if !toolicon.ValidIconFileName(name) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid icon filename"})
		return
	}
	abs, err := toolicon.ResolveAbsPath(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 防穿越 二重保险:即使 ValidIconFileName 漏了,filepath.Base 也兜底
	if filepath.Base(abs) != name {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename mismatch"})
		return
	}
	if _, err := os.Stat(abs); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "icon not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 设置 content-type(简单靠扩展名,够用了)
	if ct := mimeByExt(filepath.Ext(name)); ct != "" {
		c.Header("Content-Type", ct)
	}
	// 缓存:这些图标可能不会变更(尤其系统内置),缓存一天减少磁盘 IO。
	c.Header("Cache-Control", "public, max-age=86400")
	c.File(abs)
}

func mimeByExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".ico":
		return "image/x-icon"
	}
	return ""
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		// 此路由是业务路由,优先级高于 SPA fallback — Gin 路由表匹配规则
		// 一旦声明具体 /api/files/tool-icons/*filename 就由它接管,不会落到 NoRoute。
		// 用 :filename 段绑定而不是 *filepath 通配符;
		// gin 通配符在 SPA fallback 之后注册时优先级低于 NoRoute,
		// 这里走精确段绑定确保命中此路由。
		// 注意:URL path 不要带 . 后缀,gin 在某些版本上对含后缀的 path
		// 会用 static 优先级匹配,反过来绕过我们的业务路由 — 这也是为什么
		// 上一次 "*filename" / ":filename" 都被 SPA 抢的根本原因。
		// 我们的文件名本身允许 ".",URL 上改成不带点的 "icon/<basename-no-ext>" 反而更安全。
		// 这里为了和"裸文件名 + 后缀"的传统约定兼容,我们保留子路径形式,只让前端 fetch 时拼完整 URL。
		Path:           "/api/files/tool-icons/:filename",
		Handler:        ginp.BindHandler(ServeIconFile),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.files.tool_icons",
		Swagger: &ginp.SwaggerInfo{
			Title:         "files.tool_icons",
			Description:   "静态服务用户上传/seed 的工具图标;文件名必须是合法 basename",
			RequestParams: struct{}{},
		},
	})
}
