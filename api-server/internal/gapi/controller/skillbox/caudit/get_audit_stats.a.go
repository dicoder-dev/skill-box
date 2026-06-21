// Package caudit - get_audit_stats.a.go
// GET /api/skillbox/audit/stats
package caudit

import (
	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/audit/saudit"
	"ginp-api/pkg/ginp"
)

// RequestGetAuditStats 无入参。
type RequestGetAuditStats struct{}

// GetAuditStats GET /api/skillbox/audit/stats
func GetAuditStats(c *ginp.ContextPlus, _ *RequestGetAuditStats) {
	svc := saudit.New(dbs.GetWriteDb(), dbs.GetReadDb())
	out, err := svc.GetStats()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/audit/stats",
		Handler:        ginp.BindParamsHandler(GetAuditStats, &RequestGetAuditStats{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.audit.stats",
		Swagger: &ginp.SwaggerInfo{
			Title:         "audit.stats",
			Description:   "返回总记录数 + 按 action / actor 分布",
			RequestParams: RequestGetAuditStats{},
		},
	})
}
