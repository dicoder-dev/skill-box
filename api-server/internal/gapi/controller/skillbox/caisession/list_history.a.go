// Package caisession - list_history.a.go
// GET /api/skillbox/ai/history/list?source_path=<...>
//
// 列出 <source_path>/.skill-box/history.json 里的全部条目;
// 不存在(没对应 skill 或 .skill-box 未创建)返空数组,不算 error。
//
// 2026-07-14 增。
package caisession

import (
	"encoding/json"
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/ai/sai"
	"ginp-api/internal/settings"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestListHistory 列表请求。
// 走 GET + query 风格(BindParamsHandler 自动从 query 拿 json tag 同名字段)。
type RequestListHistory struct {
	SourcePath string `json:"source_path" form:"source_path" query:"source_path"`
}

// ResponseListHistory 列表响应。
type ResponseListHistory struct {
	Version int                       `json:"version"`
	Items   []HistoryItemView         `json:"items"`
}

// HistoryItemView 单条(对外形态)。
//
// messages 是 json.RawMessage(完整消息数组的原始 JSON),前端按需二次解析。
// 这样 controller 不需要知道 messages 内部细节,避免依赖反转。
type HistoryItemView struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Preview  string          `json:"preview"`
	Ts       int64           `json:"ts"`
	Provider string          `json:"provider,omitempty"`
	Model    string          `json:"model,omitempty"`
	Messages json.RawMessage `json:"messages"`
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
	h, err := svc.ListHistory(req.SourcePath)
	if err != nil {
		logger.Error("ai history list: %v", err)
		switch {
		case errors.Is(err, sai.ErrEmptySourcePath):
			c.JSON(400, gin.H{"error": err.Error()})
		case errors.Is(err, sai.ErrSourcePathNotInStore):
			c.JSON(404, gin.H{"error": err.Error()})
		default:
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}

	out := ResponseListHistory{Version: h.Version, Items: make([]HistoryItemView, 0, len(h.Items))}
	for _, it := range h.Items {
		out.Items = append(out.Items, HistoryItemView{
			ID:       it.ID,
			Title:    it.Title,
			Preview:  it.Preview,
			Ts:       it.Ts,
			Provider: it.Provider,
			Model:    it.Model,
			Messages: it.Messages,
		})
	}
	c.JSON(200, out)
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
			Description:   "列 <source_path>/.skill-box/history.json 全部条目;不存在返空数组",
			RequestParams:  RequestListHistory{},
		},
	})
}
