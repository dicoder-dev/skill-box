package cmarket

import (
	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/market/smarket"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestListMarketSkills 列表请求。
type RequestListMarketSkills struct {
	SourceID uint   `json:"source_id" form:"source_id"`
	Keyword  string `json:"keyword" form:"keyword"`
	Page     int    `json:"page" form:"page"`
	Size     int    `json:"size" form:"size"`
}

// RespondListMarketSkills 响应。
type RespondListMarketSkills = smarket.ListSkillsResult

// ListMarketSkills GET /api/skillbox/market/skills
//
// 列三方市场 skill 缓存;支持按 source_id / keyword 过滤 + 分页。
// 数据来源是 market_skills 表,需要先调 refresh 拉一次。
func ListMarketSkills(c *ginp.ContextPlus, req *RequestListMarketSkills) {
	svc := newService()
	out, err := svc.ListSkills(smarket.ListSkillsQuery{
		SourceID: req.SourceID,
		Keyword:  req.Keyword,
		Page:     req.Page,
		Size:     req.Size,
	})
	if err != nil {
		logger.Error("market list: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/market/skills",
		Handler:        ginp.BindParamsHandler(ListMarketSkills, &RequestListMarketSkills{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.market.skills.list",
		Swagger: &ginp.SwaggerInfo{
			Title:         "market.skills.list",
			Description:   "列三方市场 skill 缓存(需先 refresh)",
			RequestParams: RequestListMarketSkills{},
		},
	})
}
