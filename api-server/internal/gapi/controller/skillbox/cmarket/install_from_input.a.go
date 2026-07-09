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

// RequestInstallFromInput 用户输入框下载入参(2026-07-09 增)。
//
// 与 RequestPullMarketSkillV2 的差别:不要求 caller 提前知道 source_id / remote_id;
// 把"用户原文"(slug / 详情页 URL / owner/repo@skill / GitHub URL)交给 service 解析。
//
// SourceHint:
//   - 空字符串 = auto(由后端解析 input 推断;URL 输入由域名决定,纯 slug 必须配合非空 hint)
//   - "skillhub" / "skillssh" = 强制指定 source(只对非 URL 输入生效)
//
// Scope:
//   - 空 / "global" = 全局,默认
//   - "project" = 项目级,需填 ProjectID
type RequestInstallFromInput struct {
	SourceHint string `json:"source_hint"`
	Input      string `json:"input"`
	Scope      string `json:"scope"`
	ProjectID  uint   `json:"project_id"`
}

// RespondInstallFromInput 落盘结果(2026-07-09 增)。
type RespondInstallFromInput = smarket.InstallFromInputResult

// InstallFromInput POST /api/skillbox/market/install-from-input
//
// 用户在 MarketView 输入框粘贴 skill slug / 详情页 URL → 后端解析 → 下载 → 落 store。
//
// 错误处理:
//   - ErrInvalidInput → 400(用户输入无法识别,前端可保留输入让用户重试)
//   - ErrSourceNotFound / ErrSkillNotFound → 404
//   - 其它 → 500(下载/写盘失败)
//
// 2026-07-09 增:首次进入市场时前端不一定先调过 ListSources,
// 后端 Service 层兜底自 seed 默认源(skillhub / skills.sh),保证首次 install 也能跑通。
func InstallFromInput(c *ginp.ContextPlus, req *RequestInstallFromInput) {
	if req == nil || req.Input == "" {
		c.JSON(400, gin.H{"error": "input 必填"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	out, err := newServiceV2().InstallFromInput(ctx, &smarket.InstallFromInputInput{
		SourceHint: req.SourceHint,
		Input:      req.Input,
		Scope:      req.Scope,
		ProjectID:  req.ProjectID,
	})
	if err != nil {
		switch {
		case errors.Is(err, smarket.ErrInvalidInput):
			c.JSON(400, gin.H{"error": err.Error()})
		case errors.Is(err, smarket.ErrSourceNotFound):
			c.JSON(404, gin.H{"error": err.Error()})
		default:
			logger.Error("market install from input: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/market/install-from-input",
		Handler:        ginp.BindParamsHandler(InstallFromInput, &RequestInstallFromInput{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.market.install_from_input",
		Swagger: &ginp.SwaggerInfo{
			Title:         "market.install_from_input",
			Description:   "市场输入框一键拉取:用户原文(slug / 详情页 URL / GitHub URL)→ 解析 source+remoteID → 下载 → 落 store",
			RequestParams: RequestInstallFromInput{},
		},
	})
}