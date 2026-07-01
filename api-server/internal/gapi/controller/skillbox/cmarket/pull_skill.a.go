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

// RequestPullMarketSkill 拉取到 store 的请求(2026-07-01 改名:RequestInstallMarketSkill → RequestPullMarketSkill)。
type RequestPullMarketSkill struct {
	SourceID  uint   `json:"source_id"`
	RemoteID  string `json:"remote_id"`
	Scope     string `json:"scope"`
	ProjectID uint   `json:"project_id"`
}

// RequestInstallMarketSkill 旧名 alias(2026-07-01 deprecated)。
type RequestInstallMarketSkill = RequestPullMarketSkill

// RespondPullMarketSkill 拉取到 store 的响应(2026-07-01 改名:RespondInstallMarketSkill → RespondPullMarketSkill)。
type RespondPullMarketSkill = smarket.PullResult

// RespondInstallMarketSkill 旧名 alias(2026-07-01 deprecated)。
type RespondInstallMarketSkill = RespondPullMarketSkill

// PullMarketSkill POST /api/skillbox/market/install
//
// 把三方源里某个 skill 拉取到本地 store(scope=global/project),返回创建的 skill 行。
// 内部:orchestrator.DownloadFromSource 拿 canonical → sskill.Service.Create 写盘 + 写库。
//
// 2026-07-01 改名:InstallMarketSkill → PullMarketSkill。HTTP 路径 /install 保留
// 不变(向后兼容);PermissionName 改 pull,install 同步留 alias。
func PullMarketSkill(c *ginp.ContextPlus, req *RequestPullMarketSkill) {
	svc := newService()
	if req.SourceID == 0 || req.RemoteID == "" {
		c.JSON(400, gin.H{"error": "source_id / remote_id 必填"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	out, err := svc.Pull(ctx, &smarket.PullInput{
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
			logger.Error("market pull: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	// 2026-06-30 增:旧 install 端点标 deprecated,响应头提示前端改用 /install-v2。
	// 行为不变(只写盘不 apply),保留向后兼容。
	c.Header("X-Deprecated", "use /api/skillbox/market/install-v2")
	c.JSON(200, out)
}

// InstallMarketSkill 旧名 alias(2026-07-01 deprecated),新代码请用 PullMarketSkill。
func InstallMarketSkill(c *ginp.ContextPlus, req *RequestInstallMarketSkill) {
	PullMarketSkill(c, (*RequestPullMarketSkill)(req))
}

func init() {
	// 2026-07-01 改:Path 保留 /install 兼容;PermissionName 用 pull。
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/market/install",
		Handler:        ginp.BindParamsHandler(PullMarketSkill, &RequestPullMarketSkill{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.market.pull",
		Swagger: &ginp.SwaggerInfo{
			Title:         "market.pull",
			Description:   "把三方 market skill 拉取到本地 skill-box store(只写盘不 apply,推荐用 /install-v2 一站式)",
			RequestParams: RequestPullMarketSkill{},
		},
	})
}
