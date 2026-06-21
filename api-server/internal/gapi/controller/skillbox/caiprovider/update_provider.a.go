// Package caiprovider - update_provider.a.go
// POST /api/skillbox/ai/providers/update
package caiprovider

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/ai/sai"
	"ginp-api/internal/settings"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestUpdateProvider 更新 provider(PATCH 语义:空字段不改)。
// 改名时把旧 api key 一起迁到新 name(由 service.Update 负责)。
type RequestUpdateProvider struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
}

// UpdateProvider POST /api/skillbox/ai/providers/update
func UpdateProvider(c *ginp.ContextPlus, req *RequestUpdateProvider) {
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	mgr := sai.NewManager(st)
	svc := sai.New(dbs.GetWriteDb(), dbs.GetReadDb(), st, mgr)
	in := &entity.AIProvider{
		Name:     req.Name,
		Kind:     req.Kind,
		BaseURL:  req.BaseURL,
		Model:    req.Model,
		Priority: req.Priority,
		Enabled:  req.Enabled,
	}
	out, err := svc.Update(req.ID, in)
	if err != nil {
		switch {
		case errors.Is(err, sai.ErrNotFound):
			c.JSON(404, gin.H{"error": "not found"})
		case errors.Is(err, sai.ErrUnknownKind):
			c.JSON(400, gin.H{"error": err.Error()})
		default:
			// name 重复
			logger.Error("ai update: %v", err)
			c.JSON(409, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/ai/providers/update",
		Handler:        ginp.BindParamsHandler(UpdateProvider, &RequestUpdateProvider{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.ai.providers.update",
		Swagger: &ginp.SwaggerInfo{
			Title:         "ai.providers.update",
			Description:   "按 id 更新 provider;空字段不改;改名时把 api key 一起迁过去",
			RequestParams: RequestUpdateProvider{},
		},
	})
}
