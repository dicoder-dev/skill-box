// Package cagent - create_agent.a.go
// POST /api/skillbox/agents/create
//
// 新建一个用户工具(is_system 强制 false)。
package cagent

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/agent/sagent"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestCreateAgent 新建入参。
type RequestCreateAgent struct {
	AgentID      string                  `json:"agent_id"`
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

// CreateAgent POST /api/skillbox/agents/create
func CreateAgent(c *ginp.ContextPlus, req *RequestCreateAgent) {
	svc := sagent.New(dbs.GetWriteDb(), dbs.GetReadDb())
	in := &sagent.CreateInput{
		AgentID:      req.AgentID,
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
		case errors.Is(err, sagent.ErrEmptyAgentID),
			errors.Is(err, sagent.ErrEmptyDisplay),
			errors.Is(err, sagent.ErrEmptyMdi),
			errors.Is(err, sagent.ErrBadIconFile),
			errors.Is(err, sagent.ErrBadMaturity),
			errors.Is(err, sagent.ErrBadCategory),
			errors.Is(err, sagent.ErrBadScope),
			errors.Is(err, sagent.ErrEmptyPath):
			c.JSON(400, gin.H{"error": err.Error()})
		case errors.Is(err, sagent.ErrAgentIDConflict):
			c.JSON(409, gin.H{"error": err.Error()})
		default:
			logger.Error("agent create: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/agents/create",
		Handler:        ginp.BindParamsHandler(CreateAgent, &RequestCreateAgent{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.agents.create",
		Swagger: &ginp.SwaggerInfo{
			Title:         "agents.create",
			Description:   "新建工具(用户级,is_system 强制 false);前端改完建议再调 /agents/reload 让 adapter 立刻生效",
			RequestParams: RequestCreateAgent{},
		},
	})
}

func convertPaths(in []RequestPathInput) []sagent.PathInput {
	out := make([]sagent.PathInput, 0, len(in))
	for _, p := range in {
		out = append(out, sagent.PathInput{
			Scope: p.Scope, Category: p.Category, Path: p.Path, PathOrder: p.PathOrder,
		})
	}
	return out
}
