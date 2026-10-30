// Package ctool - list_tools.a.go
// GET /api/skillbox/tools
//
// 列出所有 AI 编程工具(系统 + 用户),含每条 path。
// 供前端 Settings / SkillsView 用。
package ctool

import (
	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/tool/stool"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestListTools 列表请求(无参)。
type RequestListTools struct{}

// RespondListTools 响应 = 工具视图列表。
type RespondListTools = []stool.ToolView

// ListTools GET /api/skillbox/tools
func ListTools(c *ginp.ContextPlus, _ *RequestListTools) {
	svc := stool.New(dbs.GetWriteDb(), dbs.GetReadDb())
	rows, err := svc.List()
	if err != nil {
		logger.Error("tool list: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"items": rows})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/tools",
		Handler:        ginp.BindParamsHandler(ListTools, &RequestListTools{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.tools.list",
		Swagger: &ginp.SwaggerInfo{
			Title:         "tools.list",
			Description:   "列出全部 AI 编程工具元数据(含 path),系统工具 is_system=true",
			RequestParams: RequestListTools{},
		},
	})
}
