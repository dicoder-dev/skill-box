package cskill

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestGetSkill 按 (scope, project_id, name, version) 查。
// full=true 时返回 canonical + files(给编辑器用);否则只返回 DB 元数据。
type RequestGetSkill struct {
	Scope     string `json:"scope" form:"scope"`
	ProjectID uint   `json:"project_id" form:"project_id"`
	Name      string `json:"name" form:"name"`
	Version   string `json:"version" form:"version"`
	Full      bool   `json:"full" form:"full"`
}

// GetSkill GET /api/skillbox/skills/get
func GetSkill(c *ginp.ContextPlus, req *RequestGetSkill) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(dbs.GetWriteDb(), dbs.GetReadDb(), store)
	if req.Full {
		full, gerr := svc.GetFull(req.Scope, req.Name, req.Version, req.ProjectID)
		if gerr != nil {
			if errors.Is(gerr, sskill.ErrNotFound) {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}
			if errors.Is(gerr, sskill.ErrInvalidScope) || errors.Is(gerr, sskill.ErrEmptyScope) {
				c.JSON(400, gin.H{"error": gerr.Error()})
				return
			}
			logger.Error("skill get full: %v", gerr)
			c.JSON(500, gin.H{"error": gerr.Error()})
			return
		}
		c.JSON(200, full)
		return
	}
	row, gerr := svc.Get(req.Scope, req.Name, req.Version, req.ProjectID)
	if gerr != nil {
		if errors.Is(gerr, sskill.ErrNotFound) {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		if errors.Is(gerr, sskill.ErrInvalidScope) || errors.Is(gerr, sskill.ErrEmptyScope) {
			c.JSON(400, gin.H{"error": gerr.Error()})
			return
		}
		logger.Error("skill get: %v", gerr)
		c.JSON(500, gin.H{"error": gerr.Error()})
		return
	}
	c.JSON(200, row)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/get",
		Handler:        ginp.BindParamsHandler(GetSkill, &RequestGetSkill{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.get",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.get",
			Description:   "按 (scope, project_id, name, version) 查 skill;full=true 返回 canonical + files",
			RequestParams: RequestGetSkill{},
		},
	})
}
