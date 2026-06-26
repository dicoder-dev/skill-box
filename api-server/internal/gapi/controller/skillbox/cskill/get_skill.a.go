package cskill

import (
	"errors"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestGetSkill 按 name 查;full=true 时返回 canonical + files(给编辑器用)。
type RequestGetSkill struct {
	Name string `json:"name" form:"name"`
	Full bool   `json:"full" form:"full"`
}

// GetSkill GET /api/skillbox/skills/get
func GetSkill(c *ginp.ContextPlus, req *RequestGetSkill) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(store)
	canon, gerr := svc.Get(req.Name)
	if gerr != nil {
		if errors.Is(gerr, sskill.ErrNotFound) {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		if errors.Is(gerr, sskill.ErrEmptyName) {
			c.JSON(400, gin.H{"error": gerr.Error()})
			return
		}
		logger.Error("skill get: %v", gerr)
		c.JSON(500, gin.H{"error": gerr.Error()})
		return
	}
	// source_path = skill 物理目录(store root + name),前端"在文件夹中打开"用它。
	// Canonical.SourceDir 是 adapter 扫描到的源头目录,不参与 JSON,这里单独拼一份。
	sourcePath := filepath.Join(store.Root(), canon.Manifest.Name)
	if req.Full {
		c.JSON(200, gin.H{
			"name":        canon.Manifest.Name,
			"version":     canon.Manifest.Version,
			"description": canon.Manifest.Description,
			"triggers":    canon.Manifest.Triggers,
			"author":      canon.Manifest.Author,
			"license":     canon.Manifest.License,
			"depends_on":  canon.Manifest.DependsOn,
			"source_path": sourcePath,
			"canonical":   canon,
		})
		return
	}
	c.JSON(200, gin.H{
		"name":        canon.Manifest.Name,
		"version":     canon.Manifest.Version,
		"description": canon.Manifest.Description,
		"triggers":    canon.Manifest.Triggers,
		"author":      canon.Manifest.Author,
		"license":     canon.Manifest.License,
		"depends_on":  canon.Manifest.DependsOn,
		"source_path": sourcePath,
	})
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
			Description:   "按 name 查 skill;full=true 返回 manifest + files,否则只返 manifest",
			RequestParams: RequestGetSkill{},
		},
	})
}
