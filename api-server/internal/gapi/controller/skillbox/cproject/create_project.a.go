package cproject

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/project"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestCreateProject 创建请求。
type RequestCreateProject struct {
	Name        string `json:"name"`
	Alias       string `json:"alias"`
	RootPath    string `json:"root_path"`
	Description string `json:"description"`
}

// CreateProject POST /api/skillbox/projects/create
func CreateProject(c *ginp.ContextPlus, req *RequestCreateProject) {
	svc := project.New(dbs.GetWriteDb(), dbs.GetReadDb())
	in := &entity.Project{
		Name:        req.Name,
		Alias:       req.Alias,
		RootPath:    req.RootPath,
		Description: req.Description,
	}
	out, err := svc.Create(in)
	if err != nil {
		switch {
		case errors.Is(err, project.ErrEmptyName),
			errors.Is(err, project.ErrEmptyAlias),
			errors.Is(err, project.ErrEmptyRoot):
			c.JSON(400, gin.H{"error": err.Error()})
		case errors.Is(err, project.ErrAliasExists),
			errors.Is(err, project.ErrRootExists):
			c.JSON(409, gin.H{"error": err.Error()})
		default:
			logger.Error("project create: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/projects/create",
		Handler:        ginp.BindParamsHandler(CreateProject, &RequestCreateProject{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.projects.create",
		Swagger: &ginp.SwaggerInfo{
			Title:         "projects.create",
			Description:   "新建一个项目;alias / root_path 唯一",
			RequestParams: RequestCreateProject{},
		},
	})
}
