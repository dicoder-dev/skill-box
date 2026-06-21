package cmarket

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/market/smarket"
	"ginp-api/internal/skillmarket"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestRefreshMarket 刷新请求。
type RequestRefreshMarket struct {
	SourceID uint `json:"source_id" form:"source_id"`
}

// RespondRefreshMarket 刷新响应。
type RespondRefreshMarket = skillmarket.RefreshResult

// RefreshMarket POST /api/skillbox/market/refresh
//
// 触发一个三方源的拉取(可能耗时几秒),返回 RefreshResult。
// 内部走 orchestrator.RefreshFromSource,同一 sourceID 短时间内并发会被 ignore。
func RefreshMarket(c *ginp.ContextPlus, req *RequestRefreshMarket) {
	svc := newService()
	if req.SourceID == 0 {
		c.JSON(400, gin.H{"error": "source_id 必填"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := svc.RefreshSource(ctx, req.SourceID)
	if err != nil {
		// 业务错误:source busy / not found / not impl 都返 4xx/409
		switch {
		case errors.Is(err, smarket.ErrSourceNotFound):
			c.JSON(404, gin.H{"error": err.Error()})
		case errors.Is(err, skillmarket.ErrSourceBusy):
			c.JSON(409, gin.H{"error": err.Error()})
		case errors.Is(err, skillmarket.ErrSourceDisabled):
			c.JSON(409, gin.H{"error": err.Error()})
		case errors.Is(err, skillmarket.ErrSourceNotImpl):
			c.JSON(501, gin.H{"error": err.Error()})
		default:
			logger.Error("market refresh: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/market/refresh",
		Handler:        ginp.BindParamsHandler(RefreshMarket, &RequestRefreshMarket{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.market.refresh",
		Swagger: &ginp.SwaggerInfo{
			Title:         "market.refresh",
			Description:   "触发一个三方源的拉取",
			RequestParams: RequestRefreshMarket{},
		},
	})
}
