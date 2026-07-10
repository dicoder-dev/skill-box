// Package conboarding - post_onboarding_import_global_paths.a.go
//
// POST /api/skillbox/onboarding/import-global-paths
//
// 2026-07-10 增:首页"导入技能"弹窗「全局目录」Tab 提交入口。
// 前端把用户在 ~/.agents/skills 列表里勾选的 source_path 列表 POST 过来,
// 后端逐条走 skillpkg.ImportFromPaths → store.Save,产出与 /import-local 同构的
// LocalImportResult,前端直接复用结果渲染页。
//
// 入参(JSON):
//   - paths: []string — 用户勾选的 source_path 绝对路径列表
//
// 响应(JSON):同 /import-local。
// 错误:
//   - paths 为空 → 400 + {error: "paths required"}
//   - 单条解析/落盘失败 → OK=false 出现在 results,整体仍返 200(对齐 /import-local 行为)
package conboarding

import (
	"github.com/gin-gonic/gin"
	"ginp-api/internal/skillpkg"
	"ginp-api/internal/skillstore"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestOnboardingImportGlobalPaths 全局目录批量导入入参。
type RequestOnboardingImportGlobalPaths struct {
	// Paths 用户在 ~/.agents/skills 候选列表里勾选的 source_path 绝对路径列表。
	// 至少 1 条;空 → 400。
	Paths []string `json:"paths"`
}

// PostOnboardingImportGlobalPaths 入口。
func PostOnboardingImportGlobalPaths(c *ginp.ContextPlus, req *RequestOnboardingImportGlobalPaths) {
	if len(req.Paths) == 0 {
		c.JSON(400, gin.H{"error": "paths required"})
		return
	}

	store, err := skillstore.New()
	if err != nil {
		logger.Error("import-global-paths: store init failed: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	out, err := skillpkg.ImportFromPaths(store, req.Paths)
	if err != nil {
		// 这里只会是 ErrNoSkillMD / nil store 等"前置校验错",跟 /import-local
		// 保持一致:用 400 + envelope。
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/onboarding/import-global-paths",
		Handler:        ginp.BindParamsHandler(PostOnboardingImportGlobalPaths, &RequestOnboardingImportGlobalPaths{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.onboarding.importGlobalPaths",
		Swagger: &ginp.SwaggerInfo{
			Title:         "onboarding.importGlobalPaths",
			Description:   "按 ~/.agents/skills 候选列表里勾选的 source_path 批量导入(走 skillpkg.ImportFromPaths)",
			RequestParams: RequestOnboardingImportGlobalPaths{},
		},
	})
}