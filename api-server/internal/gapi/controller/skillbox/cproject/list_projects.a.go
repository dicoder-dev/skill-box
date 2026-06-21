package cproject

import (
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/project/sproject"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RequestListProjects 列表请求。
// Page/Size 走 query;Keyword 走 query(GET) 也兼容 body(POST)。
type RequestListProjects struct {
	Page    int    `json:"page" form:"page"`
	Size    int    `json:"size" form:"size"`
	Keyword string `json:"keyword" form:"keyword"`
}

// RespondListProjects 列表响应,直接复用 service.ListResult。
type RespondListProjects = sproject.ListResult

// ListProjects GET /api/skillbox/projects?page=1&size=20&keyword=foo
func ListProjects(c *ginp.ContextPlus, req *RequestListProjects) {
	svc := sproject.New(dbs.GetWriteDb(), dbs.GetReadDb())
	out, err := svc.List(sproject.ListQuery{
		Keyword: req.Keyword,
		Page:    req.Page,
		Size:    req.Size,
	})
	if err != nil {
		logger.Error("project list: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/projects",
		Handler:        ginp.BindParamsHandler(ListProjects, &RequestListProjects{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.projects.list",
		Swagger: &ginp.SwaggerInfo{
			Title:         "projects.list",
			Description:   "列出已声明的项目,支持分页与 name 模糊匹配",
			RequestParams: RequestListProjects{},
		},
	})
}
