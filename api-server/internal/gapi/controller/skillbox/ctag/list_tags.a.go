package ctag

import (
	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/entity"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestListTags 列 tag 请求。2026-06-24 改造:用 (scope, name) 定位 skill。
type RequestListTags struct {
	Scope string `json:"scope" form:"scope"`
	Name  string `json:"name" form:"name"`
}

// RespondListTags 响应。
type RespondListTags struct {
	Items []*entity.SkillTag `json:"items"`
	Total int                `json:"total"`
}

// ListTags GET /api/skillbox/skills/tags/list
func ListTags(c *ginp.ContextPlus, req *RequestListTags) {
	if req.Scope == "" {
		req.Scope = c.Query("scope")
	}
	if req.Name == "" {
		req.Name = c.Query("name")
	}
	svc := newService()
	items, err := svc.ListTags(req.Scope, req.Name)
	if err != nil {
		logger.Error("tag list: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, RespondListTags{Items: items, Total: len(items)})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/tags/list",
		Handler:        ginp.BindParamsHandler(ListTags, &RequestListTags{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.tags.list",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.tags.list",
			Description:   "列 skill 的所有 tag(按 created_at desc);用 scope+name 定位",
			RequestParams: RequestListTags{},
		},
	})
}
