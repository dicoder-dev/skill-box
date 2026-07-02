// Package ctool - upload_icon.a.go
// POST /api/skillbox/tools/upload-icon
//
// multipart/form-data 接收单个图标文件,保存到 ~/.skill-box/tool-icons/<basename>,
// 自动用 源文件 basename + 后缀 作为目标文件名(防冲突加时间戳前缀)。
//
// 返回:{name:"claude_1719300123.png"} — 前端拿到 name 后写 tool.icon_file。
//
// 设计要点:
//   - 文件最大 256 KB(够 1024x1024 png,实际够用)
//   - 后缀白名单:png/svg/jpg/jpeg/webp/ico/gif
//   - 用 Go 标准 mime/multipart,不依赖 Gin c.SaveUploadedFile
//     (后者返回错误信息不一致,本接口要精细化错误码)
package ctool

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/tool/toolicon"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

const (
	// maxIconSize 256 KB 上限(够 1024x1024 png,常规工具 logo 远小于此)
	maxIconSize = 256 * 1024
)

// RequestUploadIcon 接收 multipart form,字段名:"file"。
type RequestUploadIcon struct {
	File *multipart.FileHeader `form:"file" binding:"required"`
}

// RespondUploadIcon 上传成功响应。
type RespondUploadIcon struct {
	Name string `json:"name"` // basename,如 "claude_1719300123.png"
	URL  string `json:"url"`  // 完整 url,如 "/api/files/tool-icons/claude_1719300123.png"
}

// UploadIcon POST /api/skillbox/tools/upload-icon
//
// Body: multipart/form-data
//   - 字段 file: 单个图片文件
//
// Response: 200 {name,url} ; 400 参数错误 ; 413 太大 ; 415 不支持后缀 ; 500 写盘失败
func UploadIcon(c *ginp.ContextPlus, req *RequestUploadIcon) {
	if req == nil || req.File == nil {
		c.JSON(400, gin.H{"error": "missing file field"})
		return
	}
	if req.File.Size > maxIconSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": fmt.Sprintf("icon too large: %d > %d bytes", req.File.Size, maxIconSize),
		})
		return
	}
	// 后缀白名单
	origName := filepath.Base(req.File.Filename)
	ext := strings.ToLower(filepath.Ext(origName))
	if !toolicon.ValidIconFileName("dummy"+ext) {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{
			"error": "unsupported extension: " + ext + " (allowed: .png/.svg/.jpg/.jpeg/.webp/.ico/.gif)",
		})
		return
	}
	// 拼目标文件名: <unix_ts>_<safe-basename>
	stem := strings.TrimSuffix(origName, filepath.Ext(origName))
	stem = sanitizeFileStem(stem)
	if stem == "" {
		stem = "icon"
	}
	targetName := fmt.Sprintf("%d_%s%s", time.Now().Unix(), stem, ext)

	// 读取并写盘
	src, err := req.File.Open()
	if err != nil {
		logger.Error("upload icon open: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer src.Close()

	data, err := io.ReadAll(io.LimitReader(src, maxIconSize+1))
	if err != nil {
		logger.Error("upload icon read: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if len(data) > maxIconSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "icon too large after read"})
		return
	}

	if _, err := toolicon.SaveBytes(targetName, data); err != nil {
		logger.Error("upload icon save: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, RespondUploadIcon{
		Name: targetName,
		URL:  "/api/files/tool-icons/" + targetName,
	})
}

// sanitizeFileStem 把 basename 的 stem 部分(去掉扩展名)收敛到安全字符:
// 只保留字母数字 / 下划线 / 中划线 / 点,其他替换为下划线。
// 长度上限 64 字符。
func sanitizeFileStem(stem string) string {
	if stem == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range stem {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "._-")
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/tools/upload-icon",
		Handler:        ginp.BindParamsHandler(UploadIcon, &RequestUploadIcon{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.tools.upload_icon",
		Swagger: &ginp.SwaggerInfo{
			Title:         "tools.upload_icon",
			Description:   "上传工具自定义图标到 ~/.skill-box/tool-icons/,返回 basename;前端再把 name 写到 tool.icon_file",
			RequestParams: RequestUploadIcon{},
		},
	})
}
