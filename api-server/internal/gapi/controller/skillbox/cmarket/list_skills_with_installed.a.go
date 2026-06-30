package cmarket

import (
	"ginp-api/internal/gapi/service/market/smarket"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RequestListMarketSkillsWithInstalled 列表请求(2026-06-30 增)。
// 与 RequestListMarketSkills 字段一致,响应多 installed map。
type RequestListMarketSkillsWithInstalled struct {
	SourceID uint   `json:"source_id" form:"source_id"`
	Keyword  string `json:"keyword" form:"keyword"`
	Page     int    `json:"page" form:"page"`
	Size     int    `json:"size" form:"size"`
}

// RespondListMarketSkillsWithInstalled 响应。
type RespondListMarketSkillsWithInstalled = smarket.ListSkillsWithInstalledResult

// ListMarketSkillsWithInstalled GET /api/skillbox/market/skills-with-installed
//
// 列三方市场 skill 缓存 + 标注本地 store 是否已存在。
// 前端用 installed 字段决定"安装 / 再装一次"按钮文案与重名弹窗。
func ListMarketSkillsWithInstalled(c *ginp.ContextPlus, req *RequestListMarketSkillsWithInstalled) {
	svc := newServiceV2()
	out, err := svc.ListSkillsWithInstalled(smarket.ListSkillsQuery{
		SourceID: req.SourceID,
		Keyword:  req.Keyword,
		Page:     req.Page,
		Size:     req.Size,
	})
	if err != nil {
		logger.Error("market list with installed: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/market/skills-with-installed",
		Handler:        ginp.BindParamsHandler(ListMarketSkillsWithInstalled, &RequestListMarketSkillsWithInstalled{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.market.skills.list_with_installed",
		Swagger: &ginp.SwaggerInfo{
			Title:         "market.skills.list_with_installed",
			Description:   "列三方市场 skill 缓存 + 标注本地 store 是否已存在(用于前端按钮文案与重名弹窗)",
			RequestParams: RequestListMarketSkillsWithInstalled{},
		},
	})
}
