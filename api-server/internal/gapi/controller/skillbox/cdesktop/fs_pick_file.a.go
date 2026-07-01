// Package cdesktop - fs_pick_file.a.go
//
// POST /api/desktop/fs/pick-file
//
// 2026-07-01 增:弹系统文件选择对话框,返回用户选中的绝对路径。
// 跟 /pick-folder 平行,本接口只用于"选文件"(目前主场景:本地 zip 导入 skill)。
//
// 入参(JSON):{ accept: [".zip", ...] } 可选,后端透传给桌面端 wails3 绑定。
// 响应:{ path: string } 取消时返 path=""。
package cdesktop

import (
	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/controller/skillbox/cdesktop/hooks"
	"ginp-api/pkg/ginp"
)

// RequestFsPickFile 弹文件选择对话框入参。
type RequestFsPickFile struct {
	// Accept 后缀过滤列表,如 []string{".zip"};空数组 = 不过滤。
	Accept []string `json:"accept"`
}

// RespondFsPickFile { path: string }。
type RespondFsPickFile struct {
	Path string `json:"path"`
}

// PostFsPickFile 入口。
func PostFsPickFile(c *ginp.ContextPlus, req *RequestFsPickFile) {
	h := hooks.Get()
	if h.FsPickFile == nil {
		// Web 端或桌面端 hook 未注入时返 501,前端降级到 <input type="file">。
		c.JSON(501, gin.H{"error": "fs.pickFile not available (no desktop hook or web mode)"})
		return
	}
	path, err := h.FsPickFile(req.Accept)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, RespondFsPickFile{Path: path})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/desktop/fs/pick-file",
		Handler:        ginp.BindParamsHandler(PostFsPickFile, &RequestFsPickFile{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "desktop.fs.pickFile",
		Swagger: &ginp.SwaggerInfo{
			Title:         "desktop.fs.pickFile",
			Description:   "弹系统文件选择对话框,返回绝对路径(取消返 path=\"\");可选 accept 后缀过滤",
			RequestParams: RequestFsPickFile{},
		},
	})
}