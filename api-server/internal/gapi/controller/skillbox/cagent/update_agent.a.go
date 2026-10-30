// Package cagent - update_agent.a.go
// POST /api/skillbox/agents/update
//
// 改一个 Agent 的元数据;系统 Agent 的 tool_id / is_system 不可改(本接口忽略这两个字段)。
package cagent

import (
	"errors"

	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/agent/sagent"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
	"github.com/gin-gonic/gin"
)

// RequestUpdateAgent 改 Agent 入参。
type RequestUpdateAgent struct {
	AgentID     string              `json:"agent_id"` // locator,不可改
	DisplayName *string             `json:"display_name,omitempty"`
	MdiIcon     *string             `json:"mdi_icon,omitempty"`
	IconFile    *string             `json:"icon_file,omitempty"`
	Maturity    *string             `json:"maturity,omitempty"`
	Note        *string             `json:"note,omitempty"`
	Enabled     *bool               `json:"enabled,omitempty"`
	SortOrder   *int                `json:"sort_order,omitempty"`
	Paths       *[]RequestPathInput `json:"paths,omitempty"`
}

// UpdateAgent POST /api/skillbox/agents/update
func UpdateAgent(c *ginp.ContextPlus, req *RequestUpdateAgent) {
	svc := sagent.New(dbs.GetWriteDb(), dbs.GetReadDb())
	in := &sagent.UpdateInput{
		AgentID:     req.AgentID,
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
		case errors.Is(err, sagent.ErrNotFound):
			c.JSON(404, gin.H{"error": err.Error()})
		case errors.Is(err, sagent.ErrEmptyDisplay),
			errors.Is(err, sagent.ErrEmptyMdi),
			errors.Is(err, sagent.ErrBadIconFile),
			errors.Is(err, sagent.ErrBadMaturity),
			errors.Is(err, sagent.ErrBadCategory),
			errors.Is(err, sagent.ErrBadScope),
			errors.Is(err, sagent.ErrEmptyPath):
			c.JSON(400, gin.H{"error": err.Error()})
		default:
			logger.Error("agent update: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/agents/update",
		Handler:        ginp.BindParamsHandler(UpdateAgent, &RequestUpdateAgent{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.agents.update",
		Swagger: &ginp.SwaggerInfo{
			Title:         "agents.update",
			Description:   "改 Agent 元数据;Paths 非 null 表示\"覆盖式替换\";改完建议再调 /agents/reload",
			RequestParams: RequestUpdateAgent{},
		},
	})
}
