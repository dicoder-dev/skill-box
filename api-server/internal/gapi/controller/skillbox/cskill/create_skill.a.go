package cskill

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/skilladapter"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestCreateSkill 创建立即可。
// Version 留空时 service 兜底 0.1.0;Manifest 字段由前端编辑器填。
type RequestCreateSkill struct {
	Scope     string                `json:"scope"`
	ProjectID uint                  `json:"project_id"`
	Name      string                `json:"name"`
	Version   string                `json:"version"`
	Manifest  skilladapter.Manifest `json:"manifest"`
	Files     []skilladapter.File   `json:"files"`
}

// CreateSkill POST /api/skillbox/skills/create
func CreateSkill(c *ginp.ContextPlus, req *RequestCreateSkill) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(store)
	canon, cerr := svc.Create(&sskill.WriteInput{
		Scope:     req.Scope,
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Version:   req.Version,
		Manifest:  req.Manifest,
		Files:     req.Files,
	})
	if cerr != nil {
		switch {
		case errors.Is(cerr, sskill.ErrEmptyName),
			errors.Is(cerr, sskill.ErrEmptyScope),
			errors.Is(cerr, sskill.ErrInvalidScope):
			c.JSON(400, gin.H{"error": cerr.Error()})
		default:
			// store.Save / 校验失败都走 422
			logger.Error("skill create: %v", cerr)
			c.JSON(422, gin.H{"error": cerr.Error()})
		}
		return
	}
	c.JSON(200, gin.H{
		"name":    canon.Manifest.Name,
		"version": canon.Manifest.Version,
	})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/create",
		Handler:        ginp.BindParamsHandler(CreateSkill, &RequestCreateSkill{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.create",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.create",
			Description:   "新建 skill;写 SKILL.md 到 ~/.skill-box/skills/<name>/SKILL.md",
			RequestParams: RequestCreateSkill{},
		},
	})
}
