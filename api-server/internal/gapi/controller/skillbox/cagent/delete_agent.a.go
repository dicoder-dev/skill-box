// Package cagent - delete_agent.a.go
// POST /api/skillbox/agents/delete
//
// 删一个用户 Agent。系统 Agent(is_system=true)不可删。
package cagent

import (
	"errors"

	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/agent/sagent"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
	"github.com/gin-gonic/gin"
)

// RequestDeleteAgent 删 Agent 入参。
type RequestDeleteAgent struct {
	AgentID string `json:"agent_id"`
}

// DeleteAgent POST /api/skillbox/agents/delete
func DeleteAgent(c *ginp.ContextPlus, req *RequestDeleteAgent) {
	svc := sagent.New(dbs.GetWriteDb(), dbs.GetReadDb())
	if err := svc.Delete(req.AgentID); err != nil {
		switch {
		case errors.Is(err, sagent.ErrNotFound):
			c.JSON(404, gin.H{"error": err.Error()})
		case errors.Is(err, sagent.ErrSystemAgentFrozen):
			c.JSON(400, gin.H{"error": err.Error()})
		default:
			logger.Error("agent delete: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/agents/delete",
		Handler:        ginp.BindParamsHandler(DeleteAgent, &RequestDeleteAgent{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.agents.delete",
		Swagger: &ginp.SwaggerInfo{
			Title:         "agents.delete",
			Description:   "删一个用户 Agent(系统 Agent is_system=true 不可删);改完建议再调 /agents/reload",
			RequestParams: RequestDeleteAgent{},
		},
	})
}
