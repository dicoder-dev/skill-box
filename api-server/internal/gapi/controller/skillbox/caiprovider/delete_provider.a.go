// Package caiprovider - delete_provider.a.go
// POST /api/skillbox/ai/providers/delete
package caiprovider

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/ai/sai"
	"ginp-api/internal/settings"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestDeleteProvider 删除入参。
type RequestDeleteProvider struct {
	ID uint `json:"id"`
}

// DeleteProvider POST /api/skillbox/ai/providers/delete
func DeleteProvider(c *ginp.ContextPlus, req *RequestDeleteProvider) {
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	mgr := sai.NewManager(st)
	svc := sai.New(dbs.GetWriteDb(), dbs.GetReadDb(), st, mgr)
	if err := svc.Delete(req.ID); err != nil {
		if errors.Is(err, sai.ErrNotFound) {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		logger.Error("ai delete: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/ai/providers/delete",
		Handler:        ginp.BindParamsHandler(DeleteProvider, &RequestDeleteProvider{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.ai.providers.delete",
		Swagger: &ginp.SwaggerInfo{
			Title:         "ai.providers.delete",
			Description:   "按 id 删 provider;同时清掉 settings 里的 api key",
			RequestParams: RequestDeleteProvider{},
		},
	})
}
