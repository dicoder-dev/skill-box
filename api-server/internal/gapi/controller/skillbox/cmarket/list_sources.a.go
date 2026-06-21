package cmarket

import (
	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/market/smarket"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestListMarketSources 列表请求(query)。
type RequestListMarketSources struct {
	IncludeDisabled bool `json:"include_disabled" form:"include_disabled"`
}

// RespondListMarketSources 列表响应。
type RespondListMarketSources = smarket.ListSourcesResult

// ListMarketSources GET /api/skillbox/market/sources
//
// 列已注册的三方源;每次返回会触发 EnsureDefaultSources(seed 默认源),
// 第一次启动后两源会出现在列表里(skillhub + skills.sh)。
func ListMarketSources(c *ginp.ContextPlus, req *RequestListMarketSources) {
	svc := newService()
	if err := svc.EnsureDefaultSources(); err != nil {
		logger.Warn("market ensure default sources: %v", err)
	}
	out, err := svc.ListSources()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/market/sources",
		Handler:        ginp.BindParamsHandler(ListMarketSources, &RequestListMarketSources{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.market.sources.list",
		Swagger: &ginp.SwaggerInfo{
			Title:         "market.sources.list",
			Description:   "列出已注册的三方市场源(skillhub / skills.sh)",
			RequestParams: RequestListMarketSources{},
		},
	})
}

// newService 工厂,统一从 dbs 取 db,避免每个 controller 重复拼装。
func newService() *smarket.Service {
	ww := dbs.GetWriteDb()
	rr := dbs.GetReadDb()
	return smarket.New(ww, rr, func() (*sskill.Service, error) {
		store, err := sskill.NewStore()
		if err != nil {
			return nil, err
		}
		return sskill.New(ww, rr, store), nil
	})
}
