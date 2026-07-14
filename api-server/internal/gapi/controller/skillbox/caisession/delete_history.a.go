// Package caisession - delete_history.a.go
// DELETE /api/skillbox/ai/history/delete?source_path=...&conv_id=...
//
// v2(2026-07-14 增):按 id 删单条;不存在幂等返 200 ok。
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

// RequestDeleteHistory 删除请求。
type RequestDeleteHistory struct {
	SourcePath string `json:"source_path" form:"source_path" query:"source_path"`
	ConvID     string `json:"conv_id" form:"conv_id" query:"conv_id"`
}

// ResponseDeleteHistory 删除响应。
type ResponseDeleteHistory struct {
	Ok bool `json:"ok"`
}

// DeleteHistory DELETE /api/skillbox/ai/history/delete
func DeleteHistory(c *ginp.ContextPlus, req *RequestDeleteHistory) {
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
	if err := svc.DeleteConv(req.SourcePath, req.ConvID); err != nil {
		logger.Error("ai history delete: %v", err)
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
	c.JSON(200, ResponseDeleteHistory{Ok: true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/ai/history/delete",
		Handler:        ginp.BindParamsHandler(DeleteHistory, &RequestDeleteHistory{}),
		// ginp 未实现 HttpDelete,前端用 POST 调(走 http.post);
		// 路径仍然带 /delete,语义清晰。
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.ai.history.delete",
		Swagger: &ginp.SwaggerInfo{
			Title:         "ai.history.delete",
			Description:   "按 conv_id 删单条;不存在幂等返 200 ok;非法 conv_id → 400。前端用 POST 调用",
			RequestParams:  RequestDeleteHistory{},
		},
	})
}
