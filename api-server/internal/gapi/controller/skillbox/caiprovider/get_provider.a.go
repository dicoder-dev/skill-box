// Package caiprovider - get_provider.a.go
// GET /api/skillbox/ai/providers/get?id=
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

// RequestGetProvider 按主键查。
type RequestGetProvider struct {
	ID uint `json:"id" form:"id"`
}

// GetProvider GET /api/skillbox/ai/providers/get?id=
func GetProvider(c *ginp.ContextPlus, req *RequestGetProvider) {
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	mgr := sai.NewManager(st)
	svc := sai.New(dbs.GetWriteDb(), dbs.GetReadDb(), st, mgr)
	row, err := svc.GetByID(req.ID)
	if err != nil {
		if errors.Is(err, sai.ErrNotFound) {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		logger.Error("ai get: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, row)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/ai/providers/get",
		Handler:        ginp.BindParamsHandler(GetProvider, &RequestGetProvider{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.ai.providers.get",
		Swagger: &ginp.SwaggerInfo{
			Title:         "ai.providers.get",
			Description:   "按主键查 provider;不带 api key",
			RequestParams: RequestGetProvider{},
		},
	})
}
