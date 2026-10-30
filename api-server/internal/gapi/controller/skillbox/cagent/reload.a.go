// Package cagent - reload.a.go
// POST /api/skillbox/agents/reload
//
// 重新从 DB 拉一次 Agent 元数据,刷 skilladapter.DefaultRegistry。
// 前端改完 Agent 后调一次,让 adapter 立刻反映新数据。
package cagent

import (
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/agent/sagent"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
	"github.com/gin-gonic/gin"
)

// RequestReload 重新加载入参(无参)。
type RequestReload struct{}

// Reload POST /api/skillbox/agents/reload
func Reload(c *ginp.ContextPlus, _ *RequestReload) {
	svc := sagent.New(dbs.GetWriteDb(), dbs.GetReadDb())
	if err := svc.Reload(); err != nil {
		logger.Error("agent reload: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/agents/reload",
		Handler:        ginp.BindParamsHandler(Reload, &RequestReload{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.agents.reload",
		Swagger: &ginp.SwaggerInfo{
			Title:         "agents.reload",
			Description:   "重新从 DB 加载 Agent 到 skilladapter.Registry;前端改完 Agent 后调一次,立刻生效",
			RequestParams: RequestReload{},
		},
	})
}
