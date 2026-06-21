package cproject

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/project/sproject"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestGetProject 按主键查。
type RequestGetProject struct {
	ID uint `json:"id" form:"id" uri:"id"`
}

// GetProject GET /api/skillbox/projects/:id
func GetProject(c *ginp.ContextPlus, req *RequestGetProject) {
	svc := sproject.New(dbs.GetWriteDb(), dbs.GetReadDb())
	out, err := svc.GetByID(req.ID)
	if err != nil {
		if errors.Is(err, sproject.ErrNotFound) {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		logger.Error("project get: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/projects/get",
		AliasePaths:    []string{"/api/skillbox/projects/:id"},
		Handler:        ginp.BindParamsHandler(GetProject, &RequestGetProject{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.projects.get",
		Swagger: &ginp.SwaggerInfo{
			Title:         "projects.get",
			Description:   "按主键查项目",
			RequestParams: RequestGetProject{},
		},
	})
}
