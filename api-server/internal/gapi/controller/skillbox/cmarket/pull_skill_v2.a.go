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

// RequestPullMarketSkillV2 一站式拉取入参(2026-07-01 改名:RequestInstallMarketSkillV2 → RequestPullMarketSkillV2)。
//
// 与 RequestPullMarketSkill 比:多 Tools / FinalName / GroupPath,内部自动 apply。
type RequestPullMarketSkillV2 struct {
	SourceID  uint     `json:"source_id"`
	RemoteID  string   `json:"remote_id"`
	Scope     string   `json:"scope"`
	ProjectID uint     `json:"project_id"`
	Tools     []string `json:"tools"`      // 可选;空数组 = 只写盘不 apply(2026-06-30 改)
	FinalName string   `json:"final_name"` // 可选;支持"另存为"重命名
	// 2026-06-30 增:分组路径(多级用 / 分隔,如 "frontend/react")。
	// 空 = 装到根(未分组);非空时写到 Manifest.GroupPath,store 落到子目录。
	GroupPath string `json:"group_path"`
}

// RequestInstallMarketSkillV2 旧名 alias(2026-07-01 deprecated),新代码请用 RequestPullMarketSkillV2。
type RequestInstallMarketSkillV2 = RequestPullMarketSkillV2

// RespondPullMarketSkillV2 响应(2026-07-01 改名:RespondInstallMarketSkillV2 → RespondPullMarketSkillV2)。
//
// 含 apply 摘要、canonical、跳过的工具列表;前端用来判断装到几个工具、哪些失败。
type RespondPullMarketSkillV2 = smarket.PullV2Result

// RespondInstallMarketSkillV2 旧名 alias(2026-07-01 deprecated),新代码请用 RespondPullMarketSkillV2。
type RespondInstallMarketSkillV2 = RespondPullMarketSkillV2

// PullMarketSkillV2 POST /api/skillbox/market/install-v2
//
// 一站式:三方 → 写盘 → apply 到 tools(scope 默认 global)。
// 失败处理:写盘失败返 4xx/5xx;apply 部分失败时 SkippedTools 列出失败工具,
// 整体仍返 200(已装到本地,只是部分工具未启用)。
//
// 2026-07-01 改名:InstallMarketSkillV2 → PullMarketSkillV2。HTTP 路径 /install-v2
// 保留不变(向后兼容);PermissionName 改 pull.v2,install.v2 同步留 alias。
func PullMarketSkillV2(c *ginp.ContextPlus, req *RequestPullMarketSkillV2) {
	if req.SourceID == 0 || req.RemoteID == "" {
		c.JSON(400, gin.H{"error": "source_id / remote_id 必填"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	out, err := newServiceV2().PullV2(ctx, &smarket.PullV2Input{
		SourceID:  req.SourceID,
		RemoteID:  req.RemoteID,
		Scope:     req.Scope,
		ProjectID: req.ProjectID,
		Tools:     req.Tools,
		FinalName: req.FinalName,
		GroupPath: req.GroupPath,
	})
	if err != nil {
		switch {
		case errors.Is(err, smarket.ErrSourceNotFound),
			errors.Is(err, smarket.ErrSkillNotFound):
			c.JSON(404, gin.H{"error": err.Error()})
		default:
			logger.Error("market pull v2: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

// InstallMarketSkillV2 旧名 alias(2026-07-01 deprecated),新代码请用 PullMarketSkillV2。
func InstallMarketSkillV2(c *ginp.ContextPlus, req *RequestInstallMarketSkillV2) {
	PullMarketSkillV2(c, (*RequestPullMarketSkillV2)(req))
}

func init() {
	// 2026-07-01 改:Path 保留 /install-v2 兼容;PermissionName 用 pull.v2。
	// 旧权限名 install.v2 在 permission system 里以 alias 形式仍命中(由 service 层 InstallV2 兜底)。
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/market/install-v2",
		Handler:        ginp.BindParamsHandler(PullMarketSkillV2, &RequestPullMarketSkillV2{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.market.pull.v2",
		Swagger: &ginp.SwaggerInfo{
			Title:         "market.pull.v2",
			Description:   "三方市场 skill 一键拉取到本地 skill-box store 并自动 apply 到工具(scope 默认 global,tools 默认本机全部 5 个工具)",
			RequestParams: RequestPullMarketSkillV2{},
		},
	})
}
