// Package caisession - list_history.a.go
// GET /api/skillbox/ai/history/list?source_path=...
//
// v2(2026-07-14 改):返 metadata-only,不读 messages。
// 字段定义见 skillboxdata.ConvMeta。
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

// RequestListHistory 列表请求。
type RequestListHistory struct {
	SourcePath string `json:"source_path" form:"source_path" query:"source_path"`
}

// ResponseListHistory v2 metadata-only 响应。
type ResponseListHistory struct {
	Items []skillboxdata.ConvMeta `json:"items"`
}

// ListHistory GET /api/skillbox/ai/history/list
func ListHistory(c *ginp.ContextPlus, req *RequestListHistory) {
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	eng := sai.NewEngine(st)
	svc := sai.New(dbs.GetWriteDb(), dbs.GetReadDb(), st, eng)

	if req.SourcePath == "" {
		c.JSON(400, gin.H{"error": "source_path is required"})
		return
	}
	items, err := svc.ListConvs(req.SourcePath)
	if err != nil {
		logger.Error("ai history list: %v", err)
		switch {
		case errors.Is(err, sai.ErrEmptySourcePath),
			errors.Is(err, sai.ErrSourcePathNotInStore):
			c.JSON(400, gin.H{"error": err.Error()})
		default:
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	if items == nil {
		items = []skillboxdata.ConvMeta{}
	}
	c.JSON(200, ResponseListHistory{Items: items})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/ai/history/list",
		Handler:        ginp.BindParamsHandler(ListHistory, &RequestListHistory{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.ai.history.list",
		Swagger: &ginp.SwaggerInfo{
			Title:         "ai.history.list",
			Description:   "列 <source_path>/.skill-box/history/ 下全部对话的 metadata(不含 messages);id/title/preview/ts/provider/model/size,按 ts desc",
			RequestParams:  RequestListHistory{},
		},
	})
}
