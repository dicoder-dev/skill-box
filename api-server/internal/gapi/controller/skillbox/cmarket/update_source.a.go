package cmarket

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/market/smarket"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestUpdateMarketSource 局部更新入参(2026-06-30 增)。
//
// enabled / config_json 都用 *T 指针:不传 = 不改;传 = 覆盖。
// 这样前端只改一个字段时不会清空另一个。
type RequestUpdateMarketSource struct {
	Enabled    *bool   `json:"enabled,omitempty"`
	ConfigJSON *string `json:"config_json,omitempty"`
}

// RespondUpdateMarketSource 响应(更新后的源)。
type RespondUpdateMarketSource = smarket.UpdateSourceInput // 这里实际返回 *entity.MarketSource,见 init 路由注解

// UpdateMarketSource POST /api/skillbox/market/sources/{id}/update
//
// 局部更新源的 enabled / config_json;前端用于"源设置"抽屉。
func UpdateMarketSource(c *ginp.ContextPlus, req *RequestUpdateMarketSource) {
	idStr := c.Param("id")
	id64, perr := strconv.ParseUint(idStr, 10, 64)
	if perr != nil || id64 == 0 {
		c.JSON(400, gin.H{"error": "id 必填且为正整数"})
		return
	}
	id := uint(id64)
	out, err := newServiceV2().UpdateSource(id, &smarket.UpdateSourceInput{
		Enabled:    req.Enabled,
		ConfigJSON: req.ConfigJSON,
	})
	if err != nil {
		if errors.Is(err, smarket.ErrSourceNotFound) {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		logger.Error("market update source: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/market/sources/:id/update",
		Handler:        ginp.BindParamsHandler(UpdateMarketSource, &RequestUpdateMarketSource{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.market.sources.update",
		Swagger: &ginp.SwaggerInfo{
			Title:         "market.sources.update",
			Description:   "局部更新源的 enabled / config_json",
			RequestParams: RequestUpdateMarketSource{},
		},
	})
}
