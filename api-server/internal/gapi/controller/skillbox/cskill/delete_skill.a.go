package cskill

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestDeleteSkill 删除入参。
type RequestDeleteSkill struct {
	Scope     string `json:"scope"`
	ProjectID uint   `json:"project_id"`
	Name      string `json:"name"`
	Version   string `json:"version"`
}

// DeleteSkill POST /api/skillbox/skills/delete
func DeleteSkill(c *ginp.ContextPlus, req *RequestDeleteSkill) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(dbs.GetWriteDb(), dbs.GetReadDb(), store)
	if derr := svc.Delete(req.Scope, req.Name, req.Version, req.ProjectID); derr != nil {
		if errors.Is(derr, sskill.ErrInvalidScope) || errors.Is(derr, sskill.ErrEmptyScope) {
			c.JSON(400, gin.H{"error": derr.Error()})
			return
		}
		logger.Error("skill delete: %v", derr)
		c.JSON(500, gin.H{"error": derr.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/delete",
		Handler:        ginp.BindParamsHandler(DeleteSkill, &RequestDeleteSkill{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.delete",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.delete",
			Description:   "按 (scope, project_id, name, version) 删 skill;幂等",
			RequestParams: RequestDeleteSkill{},
		},
	})
}
