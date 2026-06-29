package cskill

import (
	"errors"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestGetSkill 按 name 或 path 查;full=true 时返回 canonical + files(给编辑器用)。
//
// 2026-06-29 改:支持多级分组 — 用 path(完整相对路径,如 "frontend/react/use-cache")
// 替代旧版只用 name。Name 仍兼容旧调用(空时由 path 拆分得到)。
type RequestGetSkill struct {
	Name string `json:"name" form:"name"`
	Path string `json:"path" form:"path"`
	Full bool   `json:"full" form:"full"`
}

// GetSkill GET /api/skillbox/skills/get
func GetSkill(c *ginp.ContextPlus, req *RequestGetSkill) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(store)
	// 2026-06-29:解析 path。优先用 path,空时退化到 name(根下兼容)
	groupPath, name := sskill.SplitPath(req.Path)
	if name == "" {
		name = req.Name
	}
	if name == "" {
		c.JSON(400, gin.H{"error": "name or path is required"})
		return
	}
	canon, gerr := svc.GetByPath(groupPath, name)
	if gerr != nil {
		if errors.Is(gerr, sskill.ErrNotFound) {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		if errors.Is(gerr, sskill.ErrEmptyName) {
			c.JSON(400, gin.H{"error": gerr.Error()})
			return
		}
		logger.Error("skill get: %v", gerr)
		c.JSON(500, gin.H{"error": gerr.Error()})
		return
	}
	// source_path = skill 物理目录(store root + group_path + name),前端"在文件夹中打开"用它。
	// Canonical.SourceDir 是 adapter 扫描到的源头目录,不参与 JSON,这里单独拼一份。
	sourcePath := store.Root()
	if groupPath != "" {
		sourcePath = filepath.Join(sourcePath, filepath.FromSlash(groupPath))
	}
	sourcePath = filepath.Join(sourcePath, name)
	if req.Full {
		c.JSON(200, gin.H{
			"name":        canon.Manifest.Name,
			"version":     canon.Manifest.Version,
			"description": canon.Manifest.Description,
			"triggers":    canon.Manifest.Triggers,
			"author":      canon.Manifest.Author,
			"license":     canon.Manifest.License,
			"depends_on":  canon.Manifest.DependsOn,
			"source_path": sourcePath,
			"path":        req.Path,
			"group_path":  canon.Manifest.GroupPath,
			"canonical":   canon,
		})
		return
	}
	c.JSON(200, gin.H{
		"name":        canon.Manifest.Name,
		"version":     canon.Manifest.Version,
		"description": canon.Manifest.Description,
		"triggers":    canon.Manifest.Triggers,
		"author":      canon.Manifest.Author,
		"license":     canon.Manifest.License,
		"depends_on":  canon.Manifest.DependsOn,
		"source_path": sourcePath,
		"path":        req.Path,
		"group_path":  canon.Manifest.GroupPath,
	})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/get",
		Handler:        ginp.BindParamsHandler(GetSkill, &RequestGetSkill{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.get",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.get",
			Description:   "按 name 查 skill;full=true 返回 manifest + files,否则只返 manifest",
			RequestParams: RequestGetSkill{},
		},
	})
}
