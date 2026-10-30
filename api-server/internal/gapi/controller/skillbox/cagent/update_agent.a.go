// Package ctool - update_tool.a.go
// POST /api/skillbox/tools/update
//
// 改一个工具的元数据;系统工具的 tool_id / is_system 不可改(本接口忽略这两个字段)。
package ctool

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/tool/stool"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestUpdateTool 改工具入参。
type RequestUpdateTool struct {
	ToolID      string              `json:"tool_id"` // locator,不可改
	DisplayName *string             `json:"display_name,omitempty"`
	MdiIcon     *string             `json:"mdi_icon,omitempty"`
	IconFile    *string             `json:"icon_file,omitempty"`
	Maturity    *string             `json:"maturity,omitempty"`
	Note        *string             `json:"note,omitempty"`
	Enabled     *bool               `json:"enabled,omitempty"`
	SortOrder   *int                `json:"sort_order,omitempty"`
	Paths       *[]RequestPathInput `json:"paths,omitempty"`
}

// UpdateTool POST /api/skillbox/tools/update
func UpdateTool(c *ginp.ContextPlus, req *RequestUpdateTool) {
	svc := stool.New(dbs.GetWriteDb(), dbs.GetReadDb())
	in := &stool.UpdateInput{
		ToolID:      req.ToolID,
		DisplayName: req.DisplayName,
		MdiIcon:     req.MdiIcon,
		IconFile:    req.IconFile,
		Maturity:    req.Maturity,
		Note:        req.Note,
		Enabled:     req.Enabled,
		SortOrder:   req.SortOrder,
	}
	if req.Paths != nil {
		converted := convertPaths(*req.Paths)
		in.Paths = &converted
	}
	out, err := svc.Update(in)
	if err != nil {
		switch {
		case errors.Is(err, stool.ErrNotFound):
			c.JSON(404, gin.H{"error": err.Error()})
		case errors.Is(err, stool.ErrEmptyDisplay),
			errors.Is(err, stool.ErrEmptyMdi),
			errors.Is(err, stool.ErrBadIconFile),
			errors.Is(err, stool.ErrBadMaturity),
			errors.Is(err, stool.ErrBadCategory),
			errors.Is(err, stool.ErrBadScope),
			errors.Is(err, stool.ErrEmptyPath):
			c.JSON(400, gin.H{"error": err.Error()})
		default:
			logger.Error("tool update: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/tools/update",
		Handler:        ginp.BindParamsHandler(UpdateTool, &RequestUpdateTool{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.tools.update",
		Swagger: &ginp.SwaggerInfo{
			Title:         "tools.update",
			Description:   "改工具元数据;Paths 非 null 表示\"覆盖式替换\";改完建议再调 /tools/reload",
			RequestParams: RequestUpdateTool{},
		},
	})
}
