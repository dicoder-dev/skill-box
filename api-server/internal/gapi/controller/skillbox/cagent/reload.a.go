// Package ctool - reload.a.go
// POST /api/skillbox/tools/reload
//
// 重新从 DB 拉一次工具元数据,刷 skilladapter.DefaultRegistry。
// 前端改完工具后调一次,让 adapter 立刻反映新数据。
package ctool

import (
	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/tool/stool"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestReload 重新加载入参(无参)。
type RequestReload struct{}

// Reload POST /api/skillbox/tools/reload
func Reload(c *ginp.ContextPlus, _ *RequestReload) {
	svc := stool.New(dbs.GetWriteDb(), dbs.GetReadDb())
	if err := svc.Reload(); err != nil {
		logger.Error("tool reload: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/tools/reload",
		Handler:        ginp.BindParamsHandler(Reload, &RequestReload{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.tools.reload",
		Swagger: &ginp.SwaggerInfo{
			Title:         "tools.reload",
			Description:   "重新从 DB 加载工具到 skilladapter.Registry;前端改完工具后调一次,立刻生效",
			RequestParams: RequestReload{},
		},
	})
}
