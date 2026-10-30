// Package ctool - create_tool.a.go
// POST /api/skillbox/tools/create
//
// 新建一个用户工具(is_system 强制 false)。
package ctool

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/tool/stool"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestCreateTool 新建入参。
type RequestCreateTool struct {
	ToolID      string                  `json:"tool_id"`
	DisplayName string                  `json:"display_name"`
	MdiIcon     string                  `json:"mdi_icon"`
	IconFile    string                  `json:"icon_file"`
	Maturity    string                  `json:"maturity"`
	Note        string                  `json:"note"`
	Enabled     bool                    `json:"enabled"`
	SortOrder   int                     `json:"sort_order"`
	Paths       []RequestPathInput      `json:"paths"`
}

// RequestPathInput 单条 path 入参。
type RequestPathInput struct {
	Scope     string `json:"scope"`
	Category  string `json:"category"`
	Path      string `json:"path"`
	PathOrder int    `json:"path_order"`
}

// CreateTool POST /api/skillbox/tools/create
func CreateTool(c *ginp.ContextPlus, req *RequestCreateTool) {
	svc := stool.New(dbs.GetWriteDb(), dbs.GetReadDb())
	in := &stool.CreateInput{
		ToolID:      req.ToolID,
		DisplayName: req.DisplayName,
		MdiIcon:     req.MdiIcon,
		IconFile:    req.IconFile,
		Maturity:    req.Maturity,
		Note:        req.Note,
		Enabled:     req.Enabled,
		SortOrder:   req.SortOrder,
		Paths:       convertPaths(req.Paths),
	}
	out, err := svc.Create(in)
	if err != nil {
		switch {
		case errors.Is(err, stool.ErrEmptyToolID),
			errors.Is(err, stool.ErrEmptyDisplay),
			errors.Is(err, stool.ErrEmptyMdi),
			errors.Is(err, stool.ErrBadIconFile),
			errors.Is(err, stool.ErrBadMaturity),
			errors.Is(err, stool.ErrBadCategory),
			errors.Is(err, stool.ErrBadScope),
			errors.Is(err, stool.ErrEmptyPath):
			c.JSON(400, gin.H{"error": err.Error()})
		case errors.Is(err, stool.ErrToolIDConflict):
			c.JSON(409, gin.H{"error": err.Error()})
		default:
			logger.Error("tool create: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/tools/create",
		Handler:        ginp.BindParamsHandler(CreateTool, &RequestCreateTool{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.tools.create",
		Swagger: &ginp.SwaggerInfo{
			Title:         "tools.create",
			Description:   "新建工具(用户级,is_system 强制 false);前端改完建议再调 /tools/reload 让 adapter 立刻生效",
			RequestParams: RequestCreateTool{},
		},
	})
}

func convertPaths(in []RequestPathInput) []stool.PathInput {
	out := make([]stool.PathInput, 0, len(in))
	for _, p := range in {
		out = append(out, stool.PathInput{
			Scope: p.Scope, Category: p.Category, Path: p.Path, PathOrder: p.PathOrder,
		})
	}
	return out
}
