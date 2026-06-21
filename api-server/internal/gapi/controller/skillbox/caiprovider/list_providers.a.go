package caiprovider

import (
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/ai/sai"
	"ginp-api/internal/settings"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RequestListProviders 列表。
type RequestListProviders struct{}

// RespondProvider 列表行(业务层 ProviderView 复用)。
type RespondProvider = sai.ProviderView

// ListProviders GET /api/skillbox/ai/providers
func ListProviders(c *ginp.ContextPlus, _ *RequestListProviders) {
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	mgr := sai.NewManager(st)
	svc := sai.New(dbs.GetWriteDb(), dbs.GetReadDb(), st, mgr)
	rows, err := svc.ListProviders()
	if err != nil {
		logger.Error("ai list: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"items": rows})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/ai/providers",
		Handler:        ginp.BindParamsHandler(ListProviders, &RequestListProviders{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.ai.providers.list",
		Swagger: &ginp.SwaggerInfo{
			Title:         "ai.providers.list",
			Description:   "列出所有 AI provider;HasKey 标记是否已配 api key",
			RequestParams: RequestListProviders{},
		},
	})
}
