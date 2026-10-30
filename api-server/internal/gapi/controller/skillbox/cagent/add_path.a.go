// Package ctool - add_path.a.go
// POST /api/skillbox/tools/paths/add
//
// 给一个工具追加一条 path(不覆盖现有)。改完调 /tools/reload 生效。
//
// 2026-07-04 改:走 stool.Service.AddOnePath,统一在 Service 层做 (scope, category)
// 唯一性校验(单 path 模型),DB uniqueIndex 作为兜底。
package ctool

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/tool/stool"
	"ginp-api/pkg/ginp"
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
	svc := stool.New(dbs.GetWriteDb(), dbs.GetReadDb())
	out, err := svc.AddOnePath(req.ToolID, stool.PathInput{
		Scope:     req.Scope,
		Category:  req.Category,
		Path:      req.Path,
		PathOrder: req.PathOrder,
	})
	if err != nil {
		switch {
		case errors.Is(err, stool.ErrNotFound):
			c.JSON(404, gin.H{"error": "tool not found: " + req.ToolID})
		case errors.Is(err, stool.ErrPathExisted):
			c.JSON(409, gin.H{
				"error": err.Error(),
				"code":  "path_existed",
			})
		case errors.Is(err, stool.ErrBadScope),
			errors.Is(err, stool.ErrBadCategory),
			errors.Is(err, stool.ErrEmptyPath):
			c.JSON(400, gin.H{"error": err.Error()})
		default:
			c.JSON(500, gin.H{"error": err.Error()})
		}
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
			Description:   "给工具追加一条 path(不覆盖现有);同 (scope, category) 仅允许 1 条,冲突返 409。改完建议再调 /tools/reload",
			RequestParams: RequestAddPath{},
		},
	})
}