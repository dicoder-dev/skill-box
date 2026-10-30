// Package cagent - add_path.a.go
// POST /api/skillbox/agents/paths/add
//
// 给一个 Agent 追加一条 path(不覆盖现有)。改完调 /agents/reload 生效。
//
// 2026-07-04 改:走 sagent.Service.AddOnePath,统一在 Service 层做 (scope, category)
// 唯一性校验(单 path 模型),DB uniqueIndex 作为兜底。
package cagent

import (
	"errors"

	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/agent/sagent"
	"ginp-api/pkg/ginp"
	"github.com/gin-gonic/gin"
)

// RequestAddPath 加一条 path 入参。
type RequestAddPath struct {
	AgentID   string `json:"agent_id"`
	Scope     string `json:"scope"`
	Category  string `json:"category"`
	Path      string `json:"path"`
	PathOrder int    `json:"path_order"`
}

// AddPath POST /api/skillbox/agents/paths/add
func AddPath(c *ginp.ContextPlus, req *RequestAddPath) {
	svc := sagent.New(dbs.GetWriteDb(), dbs.GetReadDb())
	out, err := svc.AddOnePath(req.AgentID, sagent.PathInput{
		Scope:     req.Scope,
		Category:  req.Category,
		Path:      req.Path,
		PathOrder: req.PathOrder,
	})
	if err != nil {
		switch {
		case errors.Is(err, sagent.ErrNotFound):
			c.JSON(404, gin.H{"error": "agent not found: " + req.AgentID})
		case errors.Is(err, sagent.ErrPathExisted):
			c.JSON(409, gin.H{
				"error": err.Error(),
				"code":  "path_existed",
			})
		case errors.Is(err, sagent.ErrBadScope),
			errors.Is(err, sagent.ErrBadCategory),
			errors.Is(err, sagent.ErrEmptyPath):
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
		Path:           "/api/skillbox/agents/paths/add",
		Handler:        ginp.BindParamsHandler(AddPath, &RequestAddPath{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.agents.paths.add",
		Swagger: &ginp.SwaggerInfo{
			Title:         "agents.paths.add",
			Description:   "给 Agent 追加一条 path(不覆盖现有);同 (scope, category) 仅允许 1 条,冲突返 409。改完建议再调 /agents/reload",
			RequestParams: RequestAddPath{},
		},
	})
}
