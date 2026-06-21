package cskill

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/skilladapter"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestUpdateSkill 更新入参。
// Key 字段:(scope, project_id, name, version) → body 里给新 manifest / files。
type RequestUpdateSkill struct {
	Scope     string                `json:"scope"`
	ProjectID uint                  `json:"project_id"`
	Name      string                `json:"name"`
	Version   string                `json:"version"`
	Source    string                `json:"source"`
	SourceRef string                `json:"source_ref"`
	Manifest  skilladapter.Manifest `json:"manifest"`
	Files     []skilladapter.File   `json:"files"`
}

// UpdateSkill POST /api/skillbox/skills/update
func UpdateSkill(c *ginp.ContextPlus, req *RequestUpdateSkill) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(dbs.GetWriteDb(), dbs.GetReadDb(), store)
	out, uerr := svc.Update(req.Scope, req.Name, req.Version, req.ProjectID, &sskill.WriteInput{
		Scope:     req.Scope,
		ProjectID: req.ProjectID,
		Source:    req.Source,
		SourceRef: req.SourceRef,
		Manifest:  req.Manifest,
		Files:     req.Files,
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
	c.JSON(200, out)
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
			Description:   "按 (scope, project_id, name, version) 更新 skill 内容;version 不允许改",
			RequestParams: RequestUpdateSkill{},
		},
	})
}
