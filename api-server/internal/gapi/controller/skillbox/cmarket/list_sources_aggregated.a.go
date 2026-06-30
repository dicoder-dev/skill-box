package cmarket

import (
	"ginp-api/internal/gapi/service/market/smarket"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RespondListMarketSourcesAggregated 响应。
type RespondListMarketSourcesAggregated = smarket.ListSourcesAggregatedResult

// ListMarketSourcesAggregated GET /api/skillbox/market/sources/aggregated
//
// 列源 + 每个源在 market_skill 里的条目数 + 最近拉取时间。
// 前端源 tab 用这个,避免再单独查 sources + N 次 skill_count。
func ListMarketSourcesAggregated(c *ginp.ContextPlus) {
	svc := newServiceV2()
	if err := svc.EnsureDefaultSources(); err != nil {
		logger.Warn("market ensure default sources: %v", err)
	}
	out, err := svc.ListSourcesAggregated()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/market/sources/aggregated",
		Handler:        ginp.BindHandler(ListMarketSourcesAggregated),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.market.sources.aggregated",
		Swagger: &ginp.SwaggerInfo{
			Title:       "market.sources.aggregated",
			Description: "列源 + 每个源在 market_skill 里的条目数 + 最近拉取时间",
		},
	})
}
