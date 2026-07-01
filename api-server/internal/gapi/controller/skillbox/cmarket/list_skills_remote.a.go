package cmarket

import (
	"context"
	"time"

	"ginp-api/internal/gapi/service/market/smarket"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RequestListMarketSkillsRemote 列表请求(2026-07-01 增)。
//
// 与 RequestListMarketSkillsWithInstalled 字段一致,但内部走 adapter.Discover,
// 不读 market_skills 表 — 数据永远是三方源最新。
type RequestListMarketSkillsRemote struct {
	SourceID uint   `json:"source_id" form:"source_id"`
	Keyword  string `json:"keyword" form:"keyword"`
	Page     int    `json:"page" form:"page"`
	Size     int    `json:"size" form:"size"`
}

// RespondListMarketSkillsRemote 响应。结构与 ListSkillsWithInstalledResult 一致,
// 让前端替换调用即可,无需改 schema。
type RespondListMarketSkillsRemote = smarket.ListSkillsWithInstalledResult

// ListMarketSkillsRemote GET /api/skillbox/market/skills-remote
//
// 2026-07-01 增:走 adapter.Discover,每次都打三方源,完全不读本地缓存。
// skillhub:走 /api/skills?keyword=&pageSize=100;
// skills.sh:走 /api/audits/0..49 + substring(API 无搜索参数,只能 substring 过滤);
// installed 二次扫本地 store,不影响主列表。
//
// 2026-07-01 改:60s → 45s。
// 现状:skillhub 100 条 ~2-5s;skills.sh 拉 50 页 × 50 = 2500 条 ~5-15s。
// 45s 仍然留足余量防偶发抖动,同时让前端超时窗口对齐,避免 60s 后端超时
// 已触发而前端还在傻等。
func ListMarketSkillsRemote(c *ginp.ContextPlus, req *RequestListMarketSkillsRemote) {
	if req.SourceID == 0 {
		c.JSON(400, gin.H{"error": "source_id 必填"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	svc := newService()
	out, err := svc.ListSkillsRemote(ctx, smarket.ListSkillsQuery{
		SourceID: req.SourceID,
		Keyword:  req.Keyword,
		Page:     req.Page,
		Size:     req.Size,
	})
	if err != nil {
		logger.Error("market list remote: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/market/skills-remote",
		Handler:        ginp.BindParamsHandler(ListMarketSkillsRemote, &RequestListMarketSkillsRemote{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.market.skills.list_remote",
		Swagger: &ginp.SwaggerInfo{
			Title:         "market.skills.list_remote",
			Description:   "列三方市场 skill(纯远端,不读本地缓存)",
			RequestParams: RequestListMarketSkillsRemote{},
		},
	})
}
