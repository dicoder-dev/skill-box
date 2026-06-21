// Package caiprovider - create_provider.a.go
// POST /api/skillbox/ai/providers/create
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

// RequestCreateProvider 新建 provider。
// APIKey 字段写入后端后落到 settings 表(key 形如 ai:<name>:api_key),
// 返回前不把 APIKey 回写,避免泄露。
type RequestCreateProvider struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
}

// CreateProvider POST /api/skillbox/ai/providers/create
func CreateProvider(c *ginp.ContextPlus, req *RequestCreateProvider) {
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
	out, err := svc.Create(in)
	if err != nil {
		switch {
		case errors.Is(err, sai.ErrEmptyName),
			errors.Is(err, sai.ErrEmptyKind),
			errors.Is(err, sai.ErrUnknownKind):
			c.JSON(400, gin.H{"error": err.Error()})
		default:
			// name 重复
			logger.Error("ai create: %v", err)
			c.JSON(409, gin.H{"error": err.Error()})
		}
		return
	}
	if req.APIKey != "" {
		if kerr := svc.SetKey(out.Name, req.APIKey); kerr != nil {
			logger.Error("ai create set key: %v", kerr)
			c.JSON(500, gin.H{"error": kerr.Error()})
			return
		}
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/ai/providers/create",
		Handler:        ginp.BindParamsHandler(CreateProvider, &RequestCreateProvider{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.ai.providers.create",
		Swagger: &ginp.SwaggerInfo{
			Title:         "ai.providers.create",
			Description:   "新建 provider;可顺手设 api key",
			RequestParams: RequestCreateProvider{},
		},
	})
}
