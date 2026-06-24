package cskillapply

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skillapp/sskillapp"
	"ginp-api/internal/skillapp"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)// RequestCheckUpdates 更新检测请求。query + body 兼容。
type RequestCheckUpdates struct {
	Scope     string `json:"scope" form:"scope"`
	ProjectID uint   `json:"project_id" form:"project_id"`
}

// RespondCheckUpdates 响应。
type RespondCheckUpdates struct {
	Items   []skillapp.UpdateItem `json:"items"`
	Total   int                   `json:"total"`
	Updates int                   `json:"updates"`
}

// CheckUpdates GET /api/skillbox/skills/updates
//
// 对比本地 skill 与三方市场缓存,返回可更新列表。
// scope / project_id 可选;空时表示不过滤。
func CheckUpdates(c *ginp.ContextPlus, req *RequestCheckUpdates) {
	// 兼容 query string
	if req.ProjectID == 0 {
		if s := c.Query("project_id"); s != "" {
			if n, err := strconv.ParseUint(s, 10, 64); err == nil {
				req.ProjectID = uint(n)
			}
		}
	}
	svc := newService()
	items, err := svc.CheckUpdates(sskillapp.CheckUpdatesInput{
		Scope:     req.Scope,
		ProjectID: req.ProjectID,
	})
	if err != nil {
		logger.Error("skill check updates: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	updates := 0
	for _, it := range items {
		if it.UpdateAvailable {
			updates++
		}
	}
	c.JSON(200, RespondCheckUpdates{Items: items, Total: len(items), Updates: updates})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/updates",
		Handler:        ginp.BindParamsHandler(CheckUpdates, &RequestCheckUpdates{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.updates",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.updates",
			Description:   "对比本地 skill 与三方市场缓存,返回可更新列表",
			RequestParams: RequestCheckUpdates{},
		},
	})
}
