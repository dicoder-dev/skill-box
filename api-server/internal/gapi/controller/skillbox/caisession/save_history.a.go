// Package caisession - save_history.a.go
// POST /api/skillbox/ai/history/save
//
// 把当前活跃 AI 会话条目列表写入 "<source_path>/.skill-box/history.json",
// 作为前端 localStorage 的"权威副本"。
//
// 2026-07-14 增。
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

// RequestSaveHistory 历史保存请求。
//
// SourcePath: 磁盘绝对路径(skillstore 通过 EvalSymlinks 后的相对目录);
// 必须位于 skillstore.root 之下 + 含 SKILL.md,否则 404。
//
// Items: 历史条目列表(空数组 = 清空)。
// 单条 HistoryItem 字段定义见 skillboxdata.HistoryItem。
type RequestSaveHistory struct {
	SourcePath string                     `json:"source_path"`
	Items      []skillboxdata.HistoryItem  `json:"items"`
}

// ResponseSaveHistory 写盘结果。
type ResponseSaveHistory struct {
	Ok     bool `json:"ok"`
	Count  int  `json:"count"`
	Truncated bool `json:"truncated"` // 写盘时是否触发 FIFO 截断
}

// SaveHistory POST /api/skillbox/ai/history/save
func SaveHistory(c *ginp.ContextPlus, req *RequestSaveHistory) {
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	eng := sai.NewEngine(st)
	svc := sai.New(dbs.GetWriteDb(), dbs.GetReadDb(), st, eng)

	if req.SourcePath == "" {
		c.JSON(400, gin.H{"error": "source_path is required"})
		return
	}
	// 截断与否:简单做法 — 写完后读,对比 len。
	// 在 svc.SaveHistory 里没暴露截断信号前,以"成功且 0 ≤ count ≤ len(req.Items)"为 OK,
	// truncated 始终 false(MVP;真要标记应在 skillboxdata.WriteHistory 返 truncated 信息)。
	if err := svc.SaveHistory(req.SourcePath, req.Items); err != nil {
		logger.Error("ai history save: %v", err)
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
	c.JSON(200, ResponseSaveHistory{Ok: true, Count: len(req.Items)})
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
			Description:   "保存 AI 历史对话到 <source_path>/.skill-box/history.json;source_path 必须在 skillstore.root 下且是合法 skill",
			RequestParams:  RequestSaveHistory{},
		},
	})
}
