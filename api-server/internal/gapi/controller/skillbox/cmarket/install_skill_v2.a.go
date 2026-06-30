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

// RequestInstallMarketSkillV2 一键安装入参(2026-06-30 增)。
//
// 与 RequestInstallMarketSkill 比:多 Tools / FinalName,内部自动 apply。
type RequestInstallMarketSkillV2 struct {
	SourceID  uint     `json:"source_id"`
	RemoteID  string   `json:"remote_id"`
	Scope     string   `json:"scope"`
	ProjectID uint     `json:"project_id"`
	Tools     []string `json:"tools"`     // 可选;空 = skilladapter.AllTools
	FinalName string   `json:"final_name"` // 可选;支持"另存为"重命名
}

// RespondInstallMarketSkillV2 响应。
//
// 含 apply 摘要、canonical、跳过的工具列表;前端用来判断装到几个工具、哪些失败。
type RespondInstallMarketSkillV2 = smarket.InstallV2Result

// InstallMarketSkillV2 POST /api/skillbox/market/install-v2
//
// 一站式:三方 → 写盘 → apply 到 tools(scope 默认 global)。
// 失败处理:写盘失败返 4xx/5xx;apply 部分失败时 SkippedTools 列出失败工具,
// 整体仍返 200(已装到本地,只是部分工具未启用)。
func InstallMarketSkillV2(c *ginp.ContextPlus, req *RequestInstallMarketSkillV2) {
	if req.SourceID == 0 || req.RemoteID == "" {
		c.JSON(400, gin.H{"error": "source_id / remote_id 必填"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	out, err := newServiceV2().InstallV2(ctx, &smarket.InstallV2Input{
		SourceID:  req.SourceID,
		RemoteID:  req.RemoteID,
		Scope:     req.Scope,
		ProjectID: req.ProjectID,
		Tools:     req.Tools,
		FinalName: req.FinalName,
	})
	if err != nil {
		switch {
		case errors.Is(err, smarket.ErrSourceNotFound),
			errors.Is(err, smarket.ErrSkillNotFound):
			c.JSON(404, gin.H{"error": err.Error()})
		default:
			logger.Error("market install v2: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/market/install-v2",
		Handler:        ginp.BindParamsHandler(InstallMarketSkillV2, &RequestInstallMarketSkillV2{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.market.install.v2",
		Swagger: &ginp.SwaggerInfo{
			Title:         "market.install.v2",
			Description:   "三方市场 skill 一键装到本地 store 并自动 apply 到工具(scope 默认 global,tools 默认本机全部 5 个工具)",
			RequestParams: RequestInstallMarketSkillV2{},
		},
	})
}
