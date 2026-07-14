// Package caisession - get_history.a.go
// GET /api/skillbox/ai/history/get?source_path=...&conv_id=...
//
// v2(2026-07-14 增):按 conv_id 拉单条完整(含 messages)。
// 不存在返 404。
package caisession

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/ai/sai"
	"ginp-api/internal/settings"
	"ginp-api/internal/skillboxdata"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestGetHistory 单条取请求。
type RequestGetHistory struct {
	SourcePath string `json:"source_path" form:"source_path" query:"source_path"`
	ConvID     string `json:"conv_id" form:"conv_id" query:"conv_id"`
}

// ResponseGetHistory 单条取响应。
type ResponseGetHistory struct {
	Item skillboxdata.HistoryItem `json:"item"`
}

// GetHistory GET /api/skillbox/ai/history/get
func GetHistory(c *ginp.ContextPlus, req *RequestGetHistory) {
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	eng := sai.NewEngine(st)
	svc := sai.New(dbs.GetWriteDb(), dbs.GetReadDb(), st, eng)

	if req.SourcePath == "" {
		c.JSON(400, gin.H{"error": "source_path is required"})
		return
	}
	if req.ConvID == "" {
		c.JSON(400, gin.H{"error": "conv_id is required"})
		return
	}
	item, err := svc.GetConv(req.SourcePath, req.ConvID)
	if err != nil {
		logger.Error("ai history get: %v", err)
		switch {
		case errors.Is(err, sai.ErrEmptySourcePath),
			errors.Is(err, sai.ErrSourcePathNotInStore),
			errors.Is(err, skillboxdata.ErrInvalidConvID):
			c.JSON(400, gin.H{"error": err.Error()})
		default:
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	if item == nil {
		c.JSON(404, gin.H{"error": "conv not found", "conv_id": req.ConvID})
		return
	}
	c.JSON(200, ResponseGetHistory{Item: *item})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/ai/history/get",
		Handler:        ginp.BindParamsHandler(GetHistory, &RequestGetHistory{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.ai.history.get",
		Swagger: &ginp.SwaggerInfo{
			Title:         "ai.history.get",
			Description:   "按 conv_id 拉单条完整(含 messages);不存在 404",
			RequestParams:  RequestGetHistory{},
		},
	})
}
