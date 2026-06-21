// Package caudit - list_audit_logs.a.go
// GET /api/skillbox/audit/logs
//
// 入参: ?actor=&action=&target_type=&page=&size=
// 行为: 拉一页 audit log,按 id desc
package caudit

import (
	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/audit/saudit"
	"ginp-api/pkg/ginp"
)

// RequestListAuditLogs 列表入参(GET 走 query)。
type RequestListAuditLogs struct {
	Actor      string `form:"actor" json:"actor"`
	Action     string `form:"action" json:"action"`
	TargetType string `form:"target_type" json:"target_type"`
	Page       int    `form:"page" json:"page"`
	Size       int    `form:"size" json:"size"`
}

// ListAuditLogs GET /api/skillbox/audit/logs
func ListAuditLogs(c *ginp.ContextPlus, req *RequestListAuditLogs) {
	svc := saudit.New(dbs.GetWriteDb(), dbs.GetReadDb())
	out, err := svc.List(saudit.ListQuery{
		Actor:      req.Actor,
		Action:     req.Action,
		TargetType: req.TargetType,
		Page:       req.Page,
		Size:       req.Size,
	})
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/audit/logs",
		Handler:        ginp.BindParamsHandler(ListAuditLogs, &RequestListAuditLogs{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.audit.logs",
		Swagger: &ginp.SwaggerInfo{
			Title:         "audit.logs",
			Description:   "拉一页 audit log,按 id desc。query: actor/action/target_type/page/size",
			RequestParams: RequestListAuditLogs{},
		},
	})
}
