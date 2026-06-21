package cproject

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/project/sproject"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestUpdateProject 更新请求(部分字段语义同 PATCH)。
type RequestUpdateProject struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Alias       string `json:"alias"`
	RootPath    string `json:"root_path"`
	Description string `json:"description"`
}

// UpdateProject POST /api/skillbox/projects/update
func UpdateProject(c *ginp.ContextPlus, req *RequestUpdateProject) {
	svc := sproject.New(dbs.GetWriteDb(), dbs.GetReadDb())
	in := &entity.Project{
		Name:        req.Name,
		Alias:       req.Alias,
		RootPath:    req.RootPath,
		Description: req.Description,
	}
	out, err := svc.Update(req.ID, in)
	if err != nil {
		switch {
		case errors.Is(err, sproject.ErrNotFound):
			c.JSON(404, gin.H{"error": "not found"})
		case errors.Is(err, sproject.ErrAliasExists),
			errors.Is(err, sproject.ErrRootExists):
			c.JSON(409, gin.H{"error": err.Error()})
		default:
			logger.Error("project update: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/projects/update",
		Handler:        ginp.BindParamsHandler(UpdateProject, &RequestUpdateProject{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.projects.update",
		Swagger: &ginp.SwaggerInfo{
			Title:         "projects.update",
			Description:   "按 id 更新项目;空字段不改",
			RequestParams: RequestUpdateProject{},
		},
	})
}
