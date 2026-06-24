package cskill

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestDeleteSkill 删除入参。按 name 定位(不再需要 version)。
type RequestDeleteSkill struct {
	Name string `json:"name"`
}

// DeleteSkill POST /api/skillbox/skills/delete
func DeleteSkill(c *ginp.ContextPlus, req *RequestDeleteSkill) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(store)
	if derr := svc.Delete(req.Name); derr != nil {
		if errors.Is(derr, sskill.ErrEmptyName) {
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
			Description:   "按 name 删 skill(整个目录);幂等",
			RequestParams: RequestDeleteSkill{},
		},
	})
}
