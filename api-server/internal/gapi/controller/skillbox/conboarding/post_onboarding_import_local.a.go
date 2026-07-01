// Package conboarding - post_onboarding_import_local.a.go
//
// POST /api/skillbox/onboarding/import-local
//
// 2026-07-01 增:从本地文件夹 / 本地 zip 文件导入 skill。
// 跟 /api/skillbox/onboarding/import 的区别:这个 endpoint 不依赖"上次 scan 缓存",
// 直接按用户选择的本地路径解析 SKILL.md → 落 store。
//
// 入参(JSON):
//   - mode:   "folder" | "zip_path"
//   - path:   mode=folder 时是目录绝对路径;mode=zip_path 时是 zip 文件绝对路径
//
// 响应(JSON):跟 /api/skillbox/onboarding/import 同构(LocalImportResult),
// 前端可共用结果页。
package conboarding

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/skillpkg"
	"ginp-api/internal/skillstore"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestOnboardingImportLocal 本地导入的 JSON 入参。
type RequestOnboardingImportLocal struct {
	// Mode 决定如何解析 path:
	//   "folder"   → path 是目录,递归找 SKILL.md
	//   "zip_path" → path 是 zip 文件绝对路径
	Mode string `json:"mode"`
	// Path 磁盘绝对路径(mode=folder 时为目录;mode=zip_path 时为 zip 文件)。
	Path string `json:"path"`
}

// PostOnboardingImportLocal 入口。
func PostOnboardingImportLocal(c *ginp.ContextPlus, req *RequestOnboardingImportLocal) {
	if req.Mode == "" || req.Path == "" {
		c.JSON(400, gin.H{"error": "mode / path 必填"})
		return
	}

	store, err := skillstore.New()
	if err != nil {
		logger.Error("import-local: store init failed: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var out *skillpkg.LocalImportResult
	switch req.Mode {
	case "folder":
		out, err = skillpkg.ImportFromFolder(store, req.Path)
	case "zip_path":
		out, err = skillpkg.ImportFromZipPath(store, req.Path)
	default:
		c.JSON(400, gin.H{"error": "invalid mode: " + req.Mode})
		return
	}

	if err != nil {
		switch {
		case errors.Is(err, skillpkg.ErrNoSkillMD):
			// 400 + envelope,前端 http.post 拿到 {error, status} 走错误分支。
			c.JSON(400, gin.H{"error": err.Error()})
		default:
			logger.Error("import-local: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/onboarding/import-local",
		Handler:        ginp.BindParamsHandler(PostOnboardingImportLocal, &RequestOnboardingImportLocal{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.onboarding.importLocal",
		Swagger: &ginp.SwaggerInfo{
			Title:         "onboarding.importLocal",
			Description:   "从本地文件夹 / 本地 zip 文件导入 skill(JSON 入参,mode=folder|zip_path)",
			RequestParams: RequestOnboardingImportLocal{},
		},
	})
}