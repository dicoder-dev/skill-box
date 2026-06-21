package ctag

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/entity"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestListTags 列 tag 请求。query + body 兼容。
type RequestListTags struct {
	SkillID uint `json:"skill_id" form:"skill_id"`
}

// RespondListTags 响应。
type RespondListTags struct {
	Items []*entity.SkillTag `json:"items"`
	Total int                `json:"total"`
}

// ListTags GET /api/skillbox/skills/tags/list
//
// 列出某 skill 的所有 tag(按 created_at desc)。
func ListTags(c *ginp.ContextPlus, req *RequestListTags) {
	if req.SkillID == 0 {
		if s := c.Query("skill_id"); s != "" {
			if n, err := strconv.ParseUint(s, 10, 64); err == nil {
				req.SkillID = uint(n)
			}
		}
	}
	svc := newService()
	items, err := svc.ListTags(req.SkillID)
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
			Description:   "列 skill 的所有 tag(按 created_at desc)",
			RequestParams: RequestListTags{},
		},
	})
}
