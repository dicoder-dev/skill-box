// Package caiprovider - list_presets.a.go
// GET /api/skillbox/ai/presets
//
// 返回 aiengine.AllPresets 的快照;前端 Skills 页 AI 侧栏用它渲染 preset 按钮。
package caiprovider

import (
	"github.com/gin-gonic/gin"

	"ginp-api/internal/db/dbs"
	"ginp-api/internal/aiengine"
	"ginp-api/internal/gapi/service/ai/sai"
	"ginp-api/internal/settings"
	"ginp-api/pkg/ginp"
)

// RequestListPresets 列表请求(无参)。
type RequestListPresets struct{}

// RespondPreset 直接复用 aiengine.Preset,前端按需展示。
type RespondPreset = aiengine.Preset

// ListPresets GET /api/skillbox/ai/presets
func ListPresets(c *ginp.ContextPlus, _ *RequestListPresets) {
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	mgr := sai.NewManager(st)
	svc := sai.New(dbs.GetWriteDb(), dbs.GetReadDb(), st, mgr)
	c.JSON(200, gin.H{"items": svc.Presets()})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/ai/presets",
		Handler:        ginp.BindParamsHandler(ListPresets, &RequestListPresets{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.ai.presets.list",
		Swagger: &ginp.SwaggerInfo{
			Title:         "ai.presets.list",
			Description:   "列出内置 AI preset(优化 frontmatter / 测 description / 润色正文 / 查重复 / 安全检查)",
			RequestParams: RequestListPresets{},
		},
	})
}
