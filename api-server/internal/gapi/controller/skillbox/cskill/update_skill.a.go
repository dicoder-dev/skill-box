package cskill

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/skilladapter"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestUpdateSkill 更新入参。按 name 定位,body 里给新 manifest / files。
//
// 2026-07-12 增:DeletedPaths 是前端"明确删除"路径列表(相对 skill 根的 rel path),
// 透传给 store.Save,让 "详情目录树删文件夹/文件" 真正物理落地。
type RequestUpdateSkill struct {
	Scope        string                `json:"scope"`
	ProjectID    uint                  `json:"project_id"`
	Name         string                `json:"name"`
	Version      string                `json:"version"`
	Manifest     skilladapter.Manifest `json:"manifest"`
	Files        []skilladapter.File   `json:"files"`
	DeletedPaths []string              `json:"deleted_paths,omitempty"`
}

// UpdateSkill POST /api/skillbox/skills/update
func UpdateSkill(c *ginp.ContextPlus, req *RequestUpdateSkill) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(store)
	canon, uerr := svc.Update(req.Name, &sskill.WriteInput{
		Scope:        req.Scope,
		ProjectID:    req.ProjectID,
		Name:         req.Name,
		Version:      req.Version,
		Manifest:     req.Manifest,
		Files:        req.Files,
		DeletedPaths: req.DeletedPaths,
	})
	if uerr != nil {
		switch {
		case errors.Is(uerr, sskill.ErrNotFound):
			c.JSON(404, gin.H{"error": "not found"})
		case errors.Is(uerr, sskill.ErrEmptyName),
			errors.Is(uerr, sskill.ErrEmptyScope),
			errors.Is(uerr, sskill.ErrInvalidScope):
			c.JSON(400, gin.H{"error": uerr.Error()})
		default:
			logger.Error("skill update: %v", uerr)
			c.JSON(422, gin.H{"error": uerr.Error()})
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
		Path:           "/api/skillbox/skills/update",
		Handler:        ginp.BindParamsHandler(UpdateSkill, &RequestUpdateSkill{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.update",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.update",
			Description:   "按 name 更新 skill 内容;version 写在 SKILL.md frontmatter",
			RequestParams: RequestUpdateSkill{},
		},
	})
}
