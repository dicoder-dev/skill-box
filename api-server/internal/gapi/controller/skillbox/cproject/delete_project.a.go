package cproject

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/project"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestDeleteProject 删除请求。
type RequestDeleteProject struct {
	ID uint `json:"id"`
}

// DeleteProject POST /api/skillbox/projects/delete
func DeleteProject(c *ginp.ContextPlus, req *RequestDeleteProject) {
	svc := project.New(dbs.GetWriteDb(), dbs.GetReadDb())
	if err := svc.Delete(req.ID); err != nil {
		if errors.Is(err, project.ErrNotFound) {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		logger.Error("project delete: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/projects/delete",
		Handler:        ginp.BindParamsHandler(DeleteProject, &RequestDeleteProject{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.projects.delete",
		Swagger: &ginp.SwaggerInfo{
			Title:         "projects.delete",
			Description:   "按 id 软删除项目(后续接 skill 级联清理)",
			RequestParams: RequestDeleteProject{},
		},
	})
}
