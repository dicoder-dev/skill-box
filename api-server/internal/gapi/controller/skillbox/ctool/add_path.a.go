// Package ctool - add_path.a.go
// POST /api/skillbox/tools/paths/add
//
// 给一个工具追加一条 path(不覆盖现有)。改完调 /tools/reload 生效。
package ctool

import (
	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/model/skillbox/mtool"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestAddPath 加一条 path 入参。
type RequestAddPath struct {
	ToolID    string `json:"tool_id"`
	Scope     string `json:"scope"`
	Category  string `json:"category"`
	Path      string `json:"path"`
	PathOrder int    `json:"path_order"`
}

// AddPath POST /api/skillbox/tools/paths/add
func AddPath(c *ginp.ContextPlus, req *RequestAddPath) {
	toolM := mtool.NewModel(dbs.GetWriteDb(), dbs.GetReadDb())
	tool, err := toolM.FindByToolID(req.ToolID)
	if err != nil {
		c.JSON(404, gin.H{"error": "tool not found: " + req.ToolID})
		return
	}
	// 基础校验
	if req.Scope != "global" && req.Scope != "project" {
		c.JSON(400, gin.H{"error": "scope must be global|project"})
		return
	}
	if req.Category != "user" && req.Category != "system" {
		c.JSON(400, gin.H{"error": "category must be user|system"})
		return
	}
	if req.Path == "" {
		c.JSON(400, gin.H{"error": "path is empty"})
		return
	}
	pathM := mtool.NewToolPathModel(dbs.GetWriteDb(), dbs.GetReadDb())
	out, err := pathM.Create(&entity.ToolPath{
		ToolID:    tool.ID,
		Scope:     req.Scope,
		Category:  req.Category,
		Path:      req.Path,
		PathOrder: req.PathOrder,
	})
	if err != nil {
		// (tool_id, scope, category, path) 唯一索引冲突 → 409
		if isUniqueConflict(err) {
			c.JSON(409, gin.H{"error": "path already exists for this (scope, category)"})
			return
		}
		logger.Error("tool add path: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/tools/paths/add",
		Handler:        ginp.BindParamsHandler(AddPath, &RequestAddPath{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.tools.paths.add",
		Swagger: &ginp.SwaggerInfo{
			Title:         "tools.paths.add",
			Description:   "给工具追加一条 path(不覆盖现有);改完建议再调 /tools/reload",
			RequestParams: RequestAddPath{},
		},
	})
}

// isUniqueConflict 简单判断 GORM 报的 duplicate key 错误。
// 跨 DB 略不严谨(各驱动错信息不一),但业务层不依赖具体文本,仅做"是否 409"判定。
func isUniqueConflict(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return containsAny(msg, []string{"UNIQUE constraint failed", "Duplicate entry", "duplicate key", "unique constraint"})
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
	}
	return false
}
