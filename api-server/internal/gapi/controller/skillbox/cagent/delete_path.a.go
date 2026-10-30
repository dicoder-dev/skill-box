// Package cagent - delete_path.a.go
// POST /api/skillbox/agents/paths/delete
//
// 按 id 删一条 path。改完调 /agents/reload 生效。
package cagent

import (
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/model/skillbox/magent"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
	"github.com/gin-gonic/gin"
)

// RequestDeletePath 删 path 入参(按主键 id 删)。
type RequestDeletePath struct {
	PathID uint `json:"path_id"`
}

// DeletePath POST /api/skillbox/agents/paths/delete
func DeletePath(c *ginp.ContextPlus, req *RequestDeletePath) {
	pathM := magent.NewAgentPathModel(dbs.GetWriteDb(), dbs.GetReadDb())
	if err := pathM.DeleteByID(req.PathID); err != nil {
		logger.Error("agent delete path: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/agents/paths/delete",
		Handler:        ginp.BindParamsHandler(DeletePath, &RequestDeletePath{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.agents.paths.delete",
		Swagger: &ginp.SwaggerInfo{
			Title:         "agents.paths.delete",
			Description:   "按 id 删一条 path;改完建议再调 /agents/reload",
			RequestParams: RequestDeletePath{},
		},
	})
}
