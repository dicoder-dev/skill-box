// Package ctool - delete_path.a.go
// POST /api/skillbox/tools/paths/delete
//
// 按 id 删一条 path。改完调 /tools/reload 生效。
package ctool

import (
	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/model/skillbox/mtool"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestDeletePath 删 path 入参(按主键 id 删)。
type RequestDeletePath struct {
	PathID uint `json:"path_id"`
}

// DeletePath POST /api/skillbox/tools/paths/delete
func DeletePath(c *ginp.ContextPlus, req *RequestDeletePath) {
	pathM := mtool.NewToolPathModel(dbs.GetWriteDb(), dbs.GetReadDb())
	if err := pathM.DeleteByID(req.PathID); err != nil {
		logger.Error("tool delete path: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/tools/paths/delete",
		Handler:        ginp.BindParamsHandler(DeletePath, &RequestDeletePath{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.tools.paths.delete",
		Swagger: &ginp.SwaggerInfo{
			Title:         "tools.paths.delete",
			Description:   "按 id 删一条 path;改完建议再调 /tools/reload",
			RequestParams: RequestDeletePath{},
		},
	})
}
