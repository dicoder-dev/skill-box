// Package ctool - delete_tool.a.go
// POST /api/skillbox/tools/delete
//
// 删一个用户工具。系统工具(is_system=true)不可删。
package ctool

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/tool/stool"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestDeleteTool 删工具入参。
type RequestDeleteTool struct {
	ToolID string `json:"tool_id"`
}

// DeleteTool POST /api/skillbox/tools/delete
func DeleteTool(c *ginp.ContextPlus, req *RequestDeleteTool) {
	svc := stool.New(dbs.GetWriteDb(), dbs.GetReadDb())
	if err := svc.Delete(req.ToolID); err != nil {
		switch {
		case errors.Is(err, stool.ErrNotFound):
			c.JSON(404, gin.H{"error": err.Error()})
		case errors.Is(err, stool.ErrSystemToolFrozen):
			c.JSON(400, gin.H{"error": err.Error()})
		default:
			logger.Error("tool delete: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/tools/delete",
		Handler:        ginp.BindParamsHandler(DeleteTool, &RequestDeleteTool{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.tools.delete",
		Swagger: &ginp.SwaggerInfo{
			Title:         "tools.delete",
			Description:   "删一个用户工具(系统工具 is_system=true 不可删);改完建议再调 /tools/reload",
			RequestParams: RequestDeleteTool{},
		},
	})
}
