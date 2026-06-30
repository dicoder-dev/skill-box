package cmarket

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/market/smarket"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestInstallMarketSkill 装到 store 的请求。
type RequestInstallMarketSkill struct {
	SourceID  uint   `json:"source_id"`
	RemoteID  string `json:"remote_id"`
	Scope     string `json:"scope"`
	ProjectID uint   `json:"project_id"`
}

// RespondInstallMarketSkill 装到 store 的响应。
type RespondInstallMarketSkill = smarket.InstallResult

// InstallMarketSkill POST /api/skillbox/market/install
//
// 把三方源里某个 skill 装到本地 store(scope=global/project),返回创建的 skill 行。
// 内部:orchestrator.DownloadFromSource 拿 canonical → sskill.Service.Create 写盘 + 写库。
func InstallMarketSkill(c *ginp.ContextPlus, req *RequestInstallMarketSkill) {
	svc := newService()
	if req.SourceID == 0 || req.RemoteID == "" {
		c.JSON(400, gin.H{"error": "source_id / remote_id 必填"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	out, err := svc.Install(ctx, &smarket.InstallInput{
		SourceID:  req.SourceID,
		RemoteID:  req.RemoteID,
		Scope:     req.Scope,
		ProjectID: req.ProjectID,
	})
	if err != nil {
		switch {
		case errors.Is(err, smarket.ErrSourceNotFound):
			c.JSON(404, gin.H{"error": err.Error()})
		case errors.Is(err, smarket.ErrSkillNotFound):
			c.JSON(404, gin.H{"error": err.Error()})
		default:
			logger.Error("market install: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	// 2026-06-30 增:旧 install 端点标 deprecated,响应头提示前端改用 /install-v2。
	// 行为不变(只写盘不 apply),保留向后兼容。
	c.Header("X-Deprecated", "use /api/skillbox/market/install-v2")
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/market/install",
		Handler:        ginp.BindParamsHandler(InstallMarketSkill, &RequestInstallMarketSkill{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.market.install",
		Swagger: &ginp.SwaggerInfo{
			Title:         "market.install",
			Description:   "把三方 market skill 装到本地 store",
			RequestParams: RequestInstallMarketSkill{},
		},
	})
}
