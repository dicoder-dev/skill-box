// Package caiprovider - set_key.a.go
// POST /api/skillbox/ai/providers/key
//
// 单独设 / 改 / 清 api key,设计要点:
//   - request.key = "" 视为"清空"调用 DeleteKey
//   - request.name 必须已存在(否则返回 404),不允许新建一个空 provider
//   - 不回写 key 内容(响应里 HasKey=true 表示设置成功)
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

// RequestSetKey 设/清 api key。
type RequestSetKey struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// SetKey POST /api/skillbox/ai/providers/key
func SetKey(c *ginp.ContextPlus, req *RequestSetKey) {
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	mgr := sai.NewManager(st)
	svc := sai.New(dbs.GetWriteDb(), dbs.GetReadDb(), st, mgr)
	if _, err := svc.GetByName(req.Name); err != nil {
		if errors.Is(err, sai.ErrNotFound) {
			c.JSON(404, gin.H{"error": "provider not found"})
			return
		}
		logger.Error("ai setkey lookup: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if req.Key == "" {
		_ = svc.DeleteKey(req.Name)
		c.JSON(200, gin.H{"ok": true, "has_key": false})
		return
	}
	if err := svc.SetKey(req.Name, req.Key); err != nil {
		logger.Error("ai setkey: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true, "has_key": true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/ai/providers/key",
		Handler:        ginp.BindParamsHandler(SetKey, &RequestSetKey{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.ai.providers.key",
		Swagger: &ginp.SwaggerInfo{
			Title:         "ai.providers.key",
			Description:   "设/改/清 api key;key 写到 settings,不入 ai_providers 表;key 为空表示清空",
			RequestParams: RequestSetKey{},
		},
	})
}
