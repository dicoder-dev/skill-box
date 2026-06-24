package cskillapply

import (
	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skillapp/sskillapp"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestListApplies 列表请求。2026-06-24 改造:用 (scope, name) 定位。
type RequestListApplies struct {
	Scope string `json:"scope" form:"scope"`
	Name  string `json:"name" form:"name"`
	Tool  string `json:"tool" form:"tool"`
	Status string `json:"status" form:"status"`
	Page  int    `json:"page" form:"page"`
	Size  int    `json:"size" form:"size"`
}

// RespondListApplies 响应。
type RespondListApplies = sskillapp.ListResult

// ListApplies GET /api/skillbox/skills/apply/list
func ListApplies(c *ginp.ContextPlus, req *RequestListApplies) {
	if req.Scope == "" {
		req.Scope = c.Query("scope")
	}
	if req.Name == "" {
		req.Name = c.Query("name")
	}
	svc := newService()
	out, err := svc.List(sskillapp.ListInput{
		Scope:  req.Scope,
		Name:   req.Name,
		Tool:   req.Tool,
		Status: req.Status,
		Page:   req.Page,
		Size:   req.Size,
	})
	if err != nil {
		logger.Error("skill apply list: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/apply/list",
		Handler:        ginp.BindParamsHandler(ListApplies, &RequestListApplies{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.apply.list",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.apply.list",
			Description:   "列 apply 历史(按 applied_at desc);用 scope+name 定位",
			RequestParams: RequestListApplies{},
		},
	})
}
