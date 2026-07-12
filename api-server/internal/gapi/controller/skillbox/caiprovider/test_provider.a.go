// Package caiprovider - test_provider.a.go
// POST /api/skillbox/ai/test
//
// 用途:设置界面"测试连接"按钮专用。
//
// 入参兼容两类用法:
//   1) 给 provider_id → 用表里的元数据 + settings 已存 api key
//   2) 给裸参数(kind / base_url / model / api_key) → 直接探测,不写盘
//
// 设计要点:
//   - 不依赖 streaming:直接 Drain 收集,然后返结构化结果
//   - 30s 兜底由 service 层硬控,这里再多 5s 余量
//   - 错误原样返给前端展示(provider http body / 401 / 429 等都能看到)
package caiprovider

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/ai/sai"
	"ginp-api/internal/settings"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestTestProvider 测试连接请求。
type RequestTestProvider struct {
	ProviderID uint   `json:"provider_id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	BaseURL    string `json:"base_url"`
	Model      string `json:"model"`
	APIKey     string `json:"api_key"`
}

// TestProvider POST /api/skillbox/ai/test
func TestProvider(c *ginp.ContextPlus, req *RequestTestProvider) {
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	mgr := sai.NewManager(st)
	svc := sai.New(dbs.GetWriteDb(), dbs.GetReadDb(), st, mgr)
	result, err := svc.TestConnection(sai.TestParams{
		ProviderID: req.ProviderID,
		Name:       req.Name,
		Kind:       req.Kind,
		BaseURL:    req.BaseURL,
		Model:      req.Model,
		APIKey:     req.APIKey,
	})
	if err != nil {
		// 仅"找不到 provider"这种 404 性质的才返 404;其他都返 200 + ok=false,
		// 让前端统一走"成功/失败"分支,不在 controller 多一层判断。
		if errors.Is(err, sai.ErrNotFound) {
			c.JSON(404, gin.H{"ok": false, "message": "provider not found"})
			return
		}
		logger.Error("ai test: %v", err)
		c.JSON(200, gin.H{"ok": false, "message": err.Error()})
		return
	}
	c.JSON(200, result)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/ai/test",
		Handler:        ginp.BindParamsHandler(TestProvider, &RequestTestProvider{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.ai.providers.test",
		Swagger: &ginp.SwaggerInfo{
			Title:         "ai.providers.test",
			Description:   "探测 provider 连通性;支持已存 provider 或表单临时参数",
			RequestParams: RequestTestProvider{},
		},
	})
}
