// Package caisession - save_history.a.go
// POST /api/skillbox/ai/history/save
//
// v2(2026-07-14 改):单条 upsert 到 <source_path>/.skill-box/history/<conv-id>.json。
// 替代 v1 的"批量写 history.json"。
// body: { source_path, item: HistoryItem };每次写入是 upsert,同 conv_id 会覆盖。
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

// RequestSaveHistory v2:单条 HistoryItem。
//
// Item 内嵌 skillboxdata.HistoryItem;反序列化时 Messages 作为 json.RawMessage 透传。
// 本接口对 messages 内容形状无要求,前端可传任何 user/assistant 完整字段;
// 上层 preview 自动从首条 assistant content 算(skillboxdata.WriteConv 兜底)。
type RequestSaveHistory struct {
	SourcePath string                 `json:"source_path"`
	Item       skillboxdata.HistoryItem `json:"item"`
}

// ResponseSaveHistory 写盘结果。
type ResponseSaveHistory struct {
	Ok    bool   `json:"ok"`
	ConvID string `json:"conv_id"`
}

// ErrConvTooLargeHTTPCode 单对话 > 2MB 时返 400(2026-07-14 增)。
const ErrConvTooLargeHTTPCode = 400

// SaveHistory POST /api/skillbox/ai/history/save(单条)
func SaveHistory(c *ginp.ContextPlus, req *RequestSaveHistory) {
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	eng := sai.NewEngine(st)
	svc := sai.New(dbs.GetWriteDb(), dbs.GetReadDb(), st, eng)

	if req.SourcePath == "" {
		c.JSON(400, gin.H{"error": "source_path is required"})
		return
	}
	if req.Item.ID == "" {
		c.JSON(400, gin.H{"error": "item.id is required"})
		return
	}
	if err := svc.SaveConv(req.SourcePath, req.Item); err != nil {
		logger.Error("ai history save: %v", err)
		switch {
		case errors.Is(err, sai.ErrEmptySourcePath),
			errors.Is(err, sai.ErrSourcePathNotInStore),
			errors.Is(err, skillboxdata.ErrInvalidConvID):
			c.JSON(400, gin.H{"error": err.Error()})
		case errors.Is(err, skillboxdata.ErrConvTooLarge):
			c.JSON(ErrConvTooLargeHTTPCode, gin.H{"error": err.Error(), "code": "conv_too_large"})
		default:
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, ResponseSaveHistory{Ok: true, ConvID: req.Item.ID})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/ai/history/save",
		Handler:        ginp.BindParamsHandler(SaveHistory, &RequestSaveHistory{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.ai.history.save",
		Swagger: &ginp.SwaggerInfo{
			Title:         "ai.history.save",
			Description:   "单条 upsert AI 对话到 <source_path>/.skill-box/history/<conv-id>.json;source_path 必须在 store.root 下且含 SKILL.md;超 2MB 返 400 conv_too_large",
			RequestParams:  RequestSaveHistory{},
		},
	})
}
