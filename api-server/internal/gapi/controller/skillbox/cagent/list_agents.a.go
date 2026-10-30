// Package cagent - list_agents.a.go
// GET /api/skillbox/agents
//
// 列出所有 AI 编程Agent(系统 + 用户),含每条 path。
// 供前端 Settings / SkillsView 用。
package cagent

import (
	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/agent/sagent"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestListAgents 列表请求(无参)。
type RequestListAgents struct{}

// RespondListAgents 响应 = Agent视图列表。
type RespondListAgents = []sagent.AgentView

// ListAgents GET /api/skillbox/agents
func ListAgents(c *ginp.ContextPlus, _ *RequestListAgents) {
	svc := sagent.New(dbs.GetWriteDb(), dbs.GetReadDb())
	rows, err := svc.List()
	if err != nil {
		logger.Error("agent list: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"items": rows})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/agents",
		Handler:        ginp.BindParamsHandler(ListAgents, &RequestListAgents{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.agents.list",
		Swagger: &ginp.SwaggerInfo{
			Title:         "agents.list",
			Description:   "列出全部 AI 编程Agent元数据(含 path),系统Agent is_system=true",
			RequestParams: RequestListAgents{},
		},
	})
}
